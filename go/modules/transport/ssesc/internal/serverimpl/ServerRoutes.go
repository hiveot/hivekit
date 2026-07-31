package serverimpl

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hivekit/go/modules/transport/ssesc"
	"github.com/hiveot/hivekit/go/utils"
)

// routes for handling http server requests

// HiveOTPostResponseHRef is the HTTP path that accepts HiveOT ResponseMessage envelopes
// intended for things that post responses.
//const HiveOTPostResponseHRef = "/hiveot/response"
//const HiveOTGetSseConnectHRef = "/hiveot/sse-sc"

// CreateRoutes add the routes used in SSE-SC sub-protocol
// This is simple, one full URL to connect, and three relative paths to pass
// requests, responses and notification messages.
func (srv *SseScServerImpl) CreateRoutes(ssePath string, r chi.Router) {
	if r == nil {
		slog.Error("HiveotSseModule CreateRoutes: missing router")
		return
	}
	// SSE connection endpoint
	r.Get(ssePath, srv.onSseConnection)
	r.Post(ssesc.PostSseScNotificationPath, srv.onHttpNotificationMessage)
	r.Post(ssesc.PostSseScRequestPath, srv.onHttpRequestMessage)
	r.Post(ssesc.PostSseScResponsePath, srv.onHttpResponseMessage)
}

// DeleteRoutes removes the routes used in SSE-SC sub-protocol
func (srv *SseScServerImpl) DeleteRoutes(ssePath string, r chi.Router) {
	r.Delete(ssePath, srv.onSseConnection)
	r.Delete(ssesc.PostSseScNotificationPath, srv.onHttpNotificationMessage)
	r.Delete(ssesc.PostSseScRequestPath, srv.onHttpRequestMessage)
	r.Delete(ssesc.PostSseScResponsePath, srv.onHttpResponseMessage)
}

// onNotificationMessage handles responses sent by Things.
//
// The notification is decoded into a standard notification message and passed on
// to the registered sink.
func (srv *SseScServerImpl) onHttpNotificationMessage(w http.ResponseWriter, r *http.Request) {

	// 1. Decode the message
	rp, err := srv.httpServer.GetRequestParams(r)
	if err != nil {
		// slog.Error(err.Error())
		// utils.WriteError(w, err, 0)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	// the converter translates the payload to a NotificationMessage
	notif, err := srv.encoder.DecodeNotification(rp.ClientID, rp.Payload)
	if notif == nil || notif.AffordanceType == "" {
		err = fmt.Errorf("onHttpNotificationMessage: missing notification in payload")
		utils.WriteError(w, err, 0)
		return
	}

	// pass the notification to the sinks
	srv.ForwardNotification(notif)

	utils.WriteReply(w, true, nil, nil)
}

// onHttpRequestMessage handles request messages received over http.
//
// The request is forwarded to the registered request sink.
// If the message is processed immediately, a response is returned with the http request.
// If the message is processed asynchronously, a response is returned via the replyTo
// handler and returned via SSE.
//
// If no SSE connection is established the request fails with BadRequest. This is to notify
// the client something is wrong.
//
// Note that in case of invokeaction, the response should be an ActionStatus object.
// The handler can easily create this using req.CreateActionResponse().
func (srv *SseScServerImpl) onHttpRequestMessage(w http.ResponseWriter, r *http.Request) {

	// 1. Decode the request message
	rp, err := srv.httpServer.GetRequestParams(r)
	if err != nil {
		slog.Error(err.Error())
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	req, err := srv.encoder.DecodeRequest(rp.ClientID, rp.Payload)
	if err != nil || req.Operation == "" {
		err = fmt.Errorf("HandleRequestMessage: missing or invalid request")
		slog.Error(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	slog.Info("onHttpRequestMessage", "sender", rp.ClientID, "op", req.Operation)

	// The authenticated clientID and the cid header are required.
	if rp.ClientID == "" || rp.ConnectionID == "" {
		err = fmt.Errorf("onHttpRequestMessage: missing clientID or connectionID (cid)")
		utils.WriteError(w, err, http.StatusBadRequest)
		return
	}
	// 2. locate the SSE connection that handles the response.
	c := srv.GetConnectionByConnectionID(rp.ClientID, rp.ConnectionID)
	if c == nil {
		slog.Error("onHttpRequestMessage. No SSE connection for response.",
			"clientID", rp.ClientID, "connectionID", rp.ConnectionID,
			"correlationID", req.CorrelationID)
		err = fmt.Errorf("onHttpRequestMessage: no SSE connection")
		utils.WriteError(w, err, http.StatusBadRequest)
		return
	}

	// 3. handle requests. Forward unhandled requests to the server sink.
	sc, _ := c.(*SseScServerConnection)
	sc.OnRequest(req, srv.ForwardRequest)

	// 4. The response is sent via SSE, just confirm the request is processed
	utils.WriteReply(w, false, nil, err)
}

// onHttpResponseMessage handles responses sent by Things.
//
// As WoT doesn't support reverse connections this is only used by hiveot devices
// and services that connect as clients. In that case the server is the consumer.
//
// This receives a ResponseMessage envelope and passes it to the corresponding
// connection as if the connection received the response itself.
//
// Message flow: device POST response -> server forwards to -> connection ->
// forwards to subscriber (which is the server again, or a consumer)
//
// The message body is unmarshalled and included as the response.
func (srv *SseScServerImpl) onHttpResponseMessage(w http.ResponseWriter, r *http.Request) {

	// 1. Decode the request message
	rp, err := srv.httpServer.GetRequestParams(r)
	if err != nil {
		slog.Error(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	resp, err := srv.encoder.DecodeResponse(rp.ClientID, rp.Payload)
	if err != nil || resp.Operation == "" {
		err = fmt.Errorf("HandleResponseMessage: invalid or missing response in payload")
		slog.Error(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	// pass the response to the sinks

	// FIXME: the response should be passed to the connection, the server doesn't need an rnr
	// If a request was sent to the client (via SSE) with a callback then an RNR channel was
	// opened waiting for the response.
	c := srv.GetConnectionByConnectionID(rp.ClientID, rp.ConnectionID).(*SseScServerConnection)
	err = c.OnResponse(resp)

	///---
	// handled := srv.RnrChan.HandleResponse(resp, 0)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		utils.WriteReply(w, true, nil, err)
	}
}

// onSseConnection serves a new incoming hiveot SSE connection.
// This doesn't return until the connection is closed by either client or server.
func (srv *SseScServerImpl) onSseConnection(w http.ResponseWriter, r *http.Request) {

	//An active session is required before accepting the request. This is created on
	//authentication/login. Until then SSE cm are blocked.
	rp, err := srv.httpServer.GetRequestParams(r)

	if err != nil {
		slog.Warn("SSESC Serve. No session available yet, telling client to delay retry to 10 seconds",
			"remoteAddr", r.RemoteAddr)

		// set retry to a large number
		// see https://javascript.info/server-sent-events#reconnection
		errMsg := fmt.Sprintf("retry: %s\nevent:%s\n\n",
			"10000", "logout")
		http.Error(w, errMsg, http.StatusUnauthorized)
		//w.Write([]byte(errMsg))
		w.(http.Flusher).Flush()
		return
	}
	// add the new sse connection
	// the sse connection can only be used to *send* messages to the remote client
	// responses are received via http and passed to rnrChan handler.
	c := NewSseScServerConnection(
		rp.ClientID, rp.ConnectionID, r.RemoteAddr, r,
		srv.ForwardRequest, srv.ForwardNotification)
	c.SetTimeout(srv.respTimeout)
	err = srv.AddConnection(c)

	c.Serve(w, r)

	// finally cleanup the connection
	srv.RemoveConnection(c)
	// if m.connectHandler != nil {
	// m.connectHandler(false, c, nil)
	// }
}

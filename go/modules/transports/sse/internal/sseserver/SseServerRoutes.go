package sseserver

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	sseapi "github.com/hiveot/hivekit/go/modules/transports/sse/api"
	"github.com/hiveot/hivekit/go/msg"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hivekit/go/wot/td"
)

// routes for handling http server requests

// HiveOTPostResponseHRef is the HTTP path that accepts HiveOT ResponseMessage envelopes
// intended for agents that post responses.
//const HiveOTPostResponseHRef = "/hiveot/response"
//const HiveOTGetSseConnectHRef = "/hiveot/sse-sc"

// CreateRoutes add the routes used in SSE-SC sub-protocol
// This is simple, one endpoint to connect, and one to pass requests, using URI variables
func (m *SseTransportServer) CreateRoutes(ssePath string, r chi.Router) {
	if r == nil {
		slog.Error("HiveotSseModule CreateRoutes: missing router")
		return
	}
	// SSE connection endpoint
	r.Get(ssePath, m.onSseConnection)
	r.Post(sseapi.PostSseScNotificationPath, m.onHttpNotificationMessage)
	r.Post(sseapi.PostSseScRequestPath, m.onHttpRequestMessage)
	r.Post(sseapi.PostSseScResponsePath, m.onHttpResponseMessage)
}

// DeleteRoutes removes the routes used in SSE-SC sub-protocol
func (m *SseTransportServer) DeleteRoutes(ssePath string, r chi.Router) {
	r.Delete(ssePath, m.onSseConnection)
	r.Delete(sseapi.PostSseScNotificationPath, m.onHttpNotificationMessage)
	r.Delete(sseapi.PostSseScRequestPath, m.onHttpRequestMessage)
	r.Delete(sseapi.PostSseScResponsePath, m.onHttpResponseMessage)
}

// onNotificationMessage handles responses sent by agents.
//
// The notification is decoded into a standard notification message and passed on
// to the registered sink.
func (m *SseTransportServer) onHttpNotificationMessage(w http.ResponseWriter, r *http.Request) {

	// 1. Decode the message
	rp, err := m.httpServer.GetRequestParams(r)
	if err != nil {
		utils.WriteError(w, err, 0)
		return
	}
	// the converter translates the payload to a NotificationMessage
	notif, err := m.encoder.DecodeNotification(rp.Payload)
	if notif == nil || notif.AffordanceType == "" {
		err = fmt.Errorf("onHttpNotificationMessage: missing notification in payload")
		utils.WriteError(w, err, 0)
		return
	}
	notif.SenderID = rp.ClientID

	// pass the notification to the sinks
	m.ForwardNotification(notif)

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
func (m *SseTransportServer) onHttpRequestMessage(w http.ResponseWriter, r *http.Request) {
	var resp *msg.ResponseMessage

	// 1. Decode the request message
	rp, err := m.httpServer.GetRequestParams(r)
	if err != nil {
		slog.Error(err.Error())
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	req, err := m.encoder.DecodeRequest(rp.Payload)
	if err != nil || req.Operation == "" {
		err = fmt.Errorf("HandleRequestMessage: missing or invalid request")
		slog.Error(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	slog.Info("onHttpRequestMessage", "sender", rp.ClientID, "op", req.Operation)

	// The authenticated clientID and the cid header are required.
	req.SenderID = rp.ClientID
	if rp.ClientID == "" || rp.ConnectionID == "" {
		err = fmt.Errorf("onHttpRequestMessage: missing clientID or connectionID (cid)")
		utils.WriteError(w, err, http.StatusBadRequest)
		return
	}
	// 2. locate the SSE connection that handles the response.
	c := m.GetConnectionByConnectionID(rp.ClientID, rp.ConnectionID)
	if c == nil {
		slog.Error("onHttpRequestMessage. No SSE connection for response.",
			"clientID", rp.ClientID, "connectionID", rp.ConnectionID,
			"correlationID", req.CorrelationID)
		err = fmt.Errorf("onHttpRequestMessage: no SSE connection")
		utils.WriteError(w, err, http.StatusBadRequest)
		return
	}

	// 3. handle ping operation internally
	if req.Operation == td.HTOpPing {
		// ping responds immediately via SSE
		resp = req.CreateResponse("pong", nil)
		//
		err = c.SendResponse(resp)
		// debugger bug not stopping on WriteReply when at the bottom?
	} else {
		// server connection handles subscriptions so forward it
		sc, _ := c.(*HiveotSseServerConnection)
		handled, _ := sc.onRequestMessage(req)

		// if the connection doesnt handle the request then forward it to the
		// registered request sink, a producer running on the server.
		if !handled {
			err = m.ForwardRequest(req, c.SendResponse)
		} else {

		}
	}
	// 4. The response is sent via SSE, just confirm the request is processed
	utils.WriteReply(w, false, nil, err)
}

// onHttpResponseMessage handles responses sent by agents.
//
// As WoT doesn't support reverse connections this is only used by hiveot agents
// that connect as clients. In that case the server is the consumer.
//
// This receives a ResponseMessage envelope and passes it to the corresponding
// connection as if the connection received the response itself.
//
// Message flow: agent POST response -> server forwards to -> connection ->
// forwards to subscriber (which is the server again, or a consumer)
//
// The message body is unmarshalled and included as the response.
func (m *SseTransportServer) onHttpResponseMessage(w http.ResponseWriter, r *http.Request) {

	// 1. Decode the request message
	rp, err := m.httpServer.GetRequestParams(r)
	if err != nil {
		slog.Error(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	resp, err := m.encoder.DecodeResponse(rp.Payload)
	if err != nil || resp.Operation == "" {
		err = fmt.Errorf("HandleResponseMessage: invalid or missing response in payload")
		slog.Error(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	// pass the response to the sinks
	resp.SenderID = rp.ClientID

	// If a request was sent to the client (via SSE) with a callback then an RNR channel was
	// opened waiting for the response.
	handled := m.RnrChan.HandleResponse(resp, 0)
	if !handled {
		err := fmt.Errorf("onResponse: No response handler for request, response is lost")
		slog.Warn("onResponse: No response handler for request, response is lost",
			"correlationID", resp.CorrelationID,
			"op", resp.Operation,
			"thingID", resp.ThingID,
			"name", resp.Name)

		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		utils.WriteReply(w, true, nil, err)
	}
}

// onSseConnection serves a new incoming hiveot SSE connection.
// This doesn't return until the connection is closed by either client or server.
func (m *SseTransportServer) onSseConnection(w http.ResponseWriter, r *http.Request) {

	//An active session is required before accepting the request. This is created on
	//authentication/login. Until then SSE cm are blocked.
	rp, err := m.httpServer.GetRequestParams(r)

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
	c := NewHiveotSseConnection(
		rp.ClientID, rp.ConnectionID, r.RemoteAddr, r, m.RnrChan, m.respTimeout)

	err = m.AddConnection(c)

	c.Serve(w, r)

	// finally cleanup the connection
	m.RemoveConnection(c)
	// if m.connectHandler != nil {
	// m.connectHandler(false, c, nil)
	// }
}

package module

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hivekit/go/modules/transports/ssesc"
	"github.com/hiveot/hivekit/go/msg"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hivekit/go/wot"
)

// routes for handling http server requests

// HiveOTPostResponseHRef is the HTTP path that accepts HiveOT ResponseMessage envelopes
// intended for agents that post responses.
//const HiveOTPostResponseHRef = "/hiveot/response"
//const HiveOTGetSseConnectHRef = "/hiveot/sse-sc"

// CreateRoutes add the routes used in SSE-SC sub-protocol
// This is simple, one endpoint to connect, and one to pass requests, using URI variables
func (m *SseScModule) CreateRoutes(ssePath string, r chi.Router) {
	if r == nil {
		slog.Error("HiveotSseModule CreateRoutes: missing router")
		return
	}
	// SSE connection endpoint
	r.Get(ssePath, m.onHttpSseConnection)
	r.Post(ssesc.PostSseScNotificationPath, m.onHttpNotificationMessage)
	r.Post(ssesc.PostSseScRequestPath, m.onHttpRequestMessage)
	r.Post(ssesc.PostSseScResponsePath, m.onHttpResponseMessage)

	// Connect serves the SSE-SC protocol
	//srv.httpBasicServer.AddOps(nil, []string{SSEOpConnect},
	//	http.MethodGet, srv.ssePath, srv.Serve)
	//
	//// Handle notification messages from agents, containing a notification message envelope.
	//srv.httpBasicServer.AddOps(nil,
	//	[]string{"*"},
	//	http.MethodPost, DefaultHiveotPostNotificationHRef, srv.HandleNotificationMessage)
	//
	//// Handle request messages using a single path with URI variables.
	//srv.httpBasicServer.AddOps(nil,
	//	[]string{"*"},
	//	http.MethodPost, DefaultHiveotPostRequestHRef, srv.HandleRequestMessage)
	//
	//// Handle response messages from agents, containing a response message envelope.
	//srv.httpBasicServer.AddOps(nil,
	//	[]string{"*"},
	//	http.MethodPost, DefaultHiveotPostResponseHRef, srv.HandleResponseMessage)
}

// DeleteRoutes removes the routes used in SSE-SC sub-protocol
func (m *SseScModule) DeleteRoutes(ssePath string, r chi.Router) {
	r.Delete(ssePath, m.onHttpSseConnection)
	r.Delete(ssesc.PostSseScNotificationPath, m.onHttpNotificationMessage)
	r.Delete(ssesc.PostSseScRequestPath, m.onHttpRequestMessage)
	r.Delete(ssesc.PostSseScResponsePath, m.onHttpResponseMessage)
}

// onNotificationMessage handles responses sent by agents.
//
// The notification is decoded into a standard notification message and passed on
// to the registered sink.
func (m *SseScModule) onHttpNotificationMessage(w http.ResponseWriter, r *http.Request) {

	// 1. Decode the message
	rp, err := m.httpServer.GetRequestParams(r)
	if err != nil {
		utils.WriteError(w, err, 0)
		return
	}
	// the converter translates the payload to a NotificationMessage
	notif := m.converter.DecodeNotification(rp.Payload)
	if notif == nil || notif.Operation == "" {
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
// The request is forwarded to the registered sink.
// If the message is processed immediately, a response is returned with the http request.
// If the message is processed asynchronously, a response is returned via the replyTo
// handler and returned via SSE.
//
// If no SSE connection is established the request fails with BadRequest. This is to notify
// the client something is wrong.
//
// Note that in case of invokeaction, the response should be an ActionStatus object.
// The handler can easily create this using req.CreateActionResponse().
func (m *SseScModule) onHttpRequestMessage(w http.ResponseWriter, r *http.Request) {
	var resp *msg.ResponseMessage

	// 1. Decode the request message
	rp, err := m.httpServer.GetRequestParams(r)
	if err != nil {
		slog.Error(err.Error())
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	req := m.converter.DecodeRequest(rp.Payload)
	if req == nil || req.Operation == "" {
		err = fmt.Errorf("HandleRequestMessage: missing request in payload")
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
	if req.Operation == wot.HTOpPing {
		// ping responds immediately via SSE
		resp = req.CreateResponse("pong", nil)
		//
		err = c.SendResponse(resp)
		// debugger bug not stopping on WriteReply when at the bottom?
	} else {
		// server connection handles subscriptions so forward it
		sc, _ := c.(*HiveotSseServerConnection)
		handled, err := sc.onRequestMessage(req)

		// if the connection doesnt handle the request then forward it to the sink
		if !handled {
			err = m.ForwardRequest(req,
				func(resp *msg.ResponseMessage) error {
					err = c.SendResponse(resp)
					return err
				})
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
func (m *SseScModule) onHttpResponseMessage(w http.ResponseWriter, r *http.Request) {

	// 1. Decode the request message
	rp, err := m.httpServer.GetRequestParams(r)
	if err != nil {
		slog.Error(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	resp := m.converter.DecodeResponse(rp.Payload)
	if resp == nil || resp.Operation == "" {
		err = fmt.Errorf("HandleResponseMessage: missing response in payload")
		slog.Error(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	// pass the response to the sinks
	resp.SenderID = rp.ClientID

	// If a request was sent to the client (via SSE) with a callback then an RNR channel was
	// opened waiting for the response.
	handled := m.RnrChan.HandleResponse(resp)
	if !handled {
		// no callback waiting for the response so forward it to the sink
		err = m.ForwardResponse(resp)
	}
	if err != nil {
		// response not handled?
		slog.Error("onHttpResponseMessage: response not handled or failed in handling", "err", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		utils.WriteReply(w, true, nil, err)
	}
}

// Serve a new incoming hiveot sse connection.
// This doesn't return until the connection is closed by either client or server.
func (m *SseScModule) onHttpSseConnection(w http.ResponseWriter, r *http.Request) {

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
		rp.ClientID, rp.ConnectionID, r.RemoteAddr, r, m.RnrChan)

	// By default the server collects the requests/responses to pass it to subscribers
	// If a consumer takes over the connection (connection reversal) it will register
	// its own handlers.
	// c.SetNotificationHandler(srv.serverNotificationHandler)
	// c.SetRequestHandler(srv.serverRequestHandler)
	// c.SetResponseHandler(srv.serverResponseHandler)
	err = m.AddConnection(c)
	if m.connectHandler != nil {
		m.connectHandler(true, nil, c)
	}

	// if err != nil {
	// http.Error(w, err.Error(), http.StatusUnauthorized)
	// return
	// }
	// don't return until the connection is closed
	c.Serve(w, r)

	// finally cleanup the connection
	m.RemoveConnection(c)
	if m.connectHandler != nil {
		m.connectHandler(false, nil, c)
	}
}

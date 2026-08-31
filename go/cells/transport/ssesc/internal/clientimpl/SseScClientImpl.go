package clientimpl

import (
	"context"
	"crypto/x509"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"sync"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/cells/transport"
	"github.com/hiveot/hivekit/go/cells/transport/ssesc"
	"github.com/hiveot/hivekit/go/cells/transport/tlsclient"
	tls_client "github.com/hiveot/hivekit/go/cells/transport/tlsclient/client"
	jsoniter "github.com/json-iterator/go"
	"github.com/teris-io/shortid"
	gosse "github.com/tmaxmax/go-sse"
)

// SseScClientImpl is the http client for connecting a WoT client to a http
// server using the HiveOT http and sse sub-protocol.
//
// This implements the IConnection and IHiveCell interfaces so it can be used as
// a regular client and as a sink for other cells.
//
// This can be used by both consumers and devices.
// This is intended to be used together with an SSE return channel.
//
// The Forms needed to invoke an operations are obtained using the 'getForm'
// callback, which can be tied to a store of TD documents. The form contains the
// hiveot RequestMessage and ResponseMessage endpoints. If no form is available
// then use the default hiveot endpoints that are defined with this protocol binding.
type SseScClientImpl struct {
	*transport.TransportClientBase

	// encode/decode the request/response to the SSE messaging protocol used
	encoder transport.IMessageEncoder

	// sse cancel access
	mux sync.RWMutex

	// the request & response channel handler to match requests and responses.
	// This is used in SendRequest to wait for the response received via SSE and pass it
	// to the replyTo callbacks.
	rnrChan *msg.RnRChan

	// the sse path for the connection
	ssePath string

	// handler for closing the sse connection
	sseCancelFn context.CancelFunc

	// http2 client for posting messages
	tlsClient tlsclient.ITLSClient
}

// update the connection status and publish an notification if it differs from the last status
// a 'lost' status is ignored if the current status is set to closed as it was intentional.
func (cl *SseScClientImpl) _setConnectionStatus(newStatus api.ConnectionStatus, err error) {

	if newStatus == api.StatusLost {
		slog.Info("_setConnectionStatus SseCl client connection lost", "status", newStatus)
		// fail all outstanding RnR requests
		cl.rnrChan.CloseAll()
	}
	cl.TransportClientBase.SetConnectionStatus(newStatus, err)
}

// Close the connection with the server and set the connection status to Closed
func (cl *SseScClientImpl) Close() {
	// set status to closed and notify subscribers
	cl._setConnectionStatus(api.StatusClosed, nil)

	cl.mux.Lock()
	cancelFn := cl.sseCancelFn
	cl.sseCancelFn = nil
	cl.mux.Unlock()
	cl.tlsClient.Close()

	if cancelFn != nil {
		cancelFn()
	}
}

// Connect establishes the sse connection using the previously set credentials.
// the connection status of this client is based on the sse connection.
//
// cl._setConnectionStatus will invoked when the first ping event is received from the server.
// (go-sse doesn't have a connected callback)
func (cl *SseScClientImpl) Connect() (err error) {
	status := cl.GetConnectionStatus()
	if status == api.StatusConnected {
		return nil
	} else if status == api.StatusConnecting {
		return fmt.Errorf("Connect: busy connecting.")
	}

	if cl.ssePath == "" {
		return fmt.Errorf("connectSSE: Missing SSE path")
	}

	// configure auth for the tls client
	clientID := cl.GetClientID()
	authToken, _ := cl.GetAuthToken()
	err = cl.tlsClient.SetAuthToken(clientID, authToken)

	clientCert := cl.GetClientCert()
	if clientCert != nil {
		cl.tlsClient.SetClientCert(clientCert)
	}

	// the credentials are already set in the tlsClient
	sseCancel, err := ConnectSSE(
		cl.tlsClient,
		cl.ssePath,
		cl._setConnectionStatus,
		cl.handleSseEvent,
		cl.GetTimeout())

	cl.mux.Lock()
	cl.sseCancelFn = sseCancel
	cl.mux.Unlock()

	return err
}

// Provide the native http client used by this client
func (cl *SseScClientImpl) GetHttpClient() *http.Client {
	return cl.tlsClient.GetHttpClient()
}

// HandleNotification receives an incoming notification and sends it to the server.
// Set this as a sink of a Thing device. Do not use for consumers.
func (m *SseScClientImpl) HandleNotification(notif *msg.NotificationMessage) {

	m.SendNotification(notif)
}

// Clients receives a request
// - reconnect actions are handled here
// - other requests (like subscribe) are send to the server
func (cl *SseScClientImpl) HandleRequest(request *msg.RequestMessage, replyTo msg.ResponseHandler) error {
	if request.ThingID == cl.GetThingID() {
		if request.Operation == td.OpInvokeAction && request.Name == api.ClientConnectAction {
			err := cl.Connect()
			status := cl.GetConnectionStatus()
			resp := request.CreateResponse(status, err)
			return replyTo(resp)
		} else {
			return fmt.Errorf("HandleRequest: invalid request op='%s', name='%s'",
				request.Operation, request.Name)
		}
	}
	err := cl.SendRequest(request, replyTo)
	return err
}

// handleSSEEvent processes the push-event received from the server.
// This splits the message into notification, response and request
// requests have an operation and correlationID
// responses have no operations and a correlationID
// notifications have an operations and no correlationID
func (cl *SseScClientImpl) handleSseEvent(event gosse.Event) {
	clientID := cl.GetClientID()

	// no further processing of a ping needed
	if event.Type == ssesc.SSEPingEvent {
		return
	}

	// Use the hiveot message envelopes for request, response and notification
	switch event.Type {
	case msg.MessageTypeNotification:
		notif, err := cl.encoder.DecodeNotification("", []byte(event.Data))
		if err != nil {
			return
		}
		go cl.EmitNotification(notif)
	case msg.MessageTypeRequest:
		var err error
		req, err := cl.encoder.DecodeRequest("", []byte(event.Data))
		if err != nil {
			return
		}

		err = cl.EmitRequest(req, func(resp *msg.ResponseMessage) error {
			// return the response to the caller
			err2 := cl.SendResponse(resp)
			return err2
		})

		// responses are optional
		if err != nil {
			resp := req.CreateErrorResponse(err)
			_ = cl.SendResponse(resp)
		}

	case msg.MessageTypeResponse:
		resp, err := cl.encoder.DecodeResponse("", []byte(event.Data))
		if err != nil {
			slog.Info("handleSseEvent: Received SSE Event but decoder returns nil", "data", string(event.Data))
			return
		}

		// consumer receives a response
		// this will be 'handled' if it was waiting on its rnr channel
		handled := cl.rnrChan.HandleResponse(resp, cl.GetTimeout())

		if !handled {
			slog.Warn("handleSseEvent: No response handler for request, response is lost",
				"correlationID", resp.CorrelationID,
				"op", resp.Operation,
				"thingID", resp.ThingID,
				"name", resp.Name,
				"clientID", clientID,
			)
		} else {
			// slog.Info("SSE Response was handled in RnR",
			// "op", resp.Operation, "correlationID", resp.CorrelationID)
		}
	default:

		// all other events are intended for other use-cases such as the UI,
		// and can have a formats of event/{dThingID}/{name}
		// Attempt to deliver this for compatibility with other protocols (such has hiveoview test client)
		senderID := "" // unknown
		notif := msg.NewNotificationMessage(
			senderID, msg.AffordanceType(event.Type), "", "", event.Data)

		// don't block the receiver flow
		go cl.EmitNotification(notif)
	}
}

// Return wheter the client is still active. Status connecting or connected.
// If so, authentication and connect are not allowed. Call Close first()
func (cl *SseScClientImpl) IsRunning() bool {
	status := cl.GetConnectionStatus()

	return status == api.StatusConnected || status == api.StatusConnecting
}

// SendNotification Device posts a notification using the hiveot http/sse protocol.
func (cl *SseScClientImpl) SendNotification(msg *msg.NotificationMessage) {
	// Send as text, not binary, to avoid unmarshalling problems
	outputJSON, _ := jsoniter.MarshalToString(msg)
	_, _, err := cl.tlsClient.Post(
		ssesc.PostSseScNotificationPath, []byte(outputJSON))

	if err != nil {
		slog.Warn("SendNotification failed",
			"clientID", cl.GetClientID(),
			"err", err.Error())
	}
}

// SendRequest [Consumer] sends the RequestMessage envelope to the server
// using http. The response will be sent by the server over SSE.
//
// This uses the rnr-channel to correlate request with response and invoke replyTo.
//
// No use using forms to determine the endpoint as the response is sent via
// a single SSE return channel that the WoT specification doesn't (yet) support.
func (cl *SseScClientImpl) SendRequest(
	req *msg.RequestMessage, replyTo msg.ResponseHandler) error {

	// a correlationID is required
	if req.CorrelationID == "" {
		req.CorrelationID = shortid.MustGenerate()
	}
	// Send as text, not binary array, to avoid encoding problems when unmarshalling
	outputJSON, _ := jsoniter.MarshalToString(req)

	// If no replyTo is provided then just sent the request. The response will
	// be received async via SSE.
	if replyTo == nil {
		outputRaw, code, err := cl.tlsClient.Post(
			ssesc.PostSseScRequestPath, []byte(outputJSON))
		_ = code
		_ = outputRaw

		return err
	}

	// A response handler is provided. Invoke replyTo when the response is received
	// via sse.
	slog.Debug("HiveotSseClient.Sendrequest. Adding to RNR", "correlationID", req.CorrelationID)
	cl.rnrChan.Open(req.CorrelationID)

	outputRaw, code, err := cl.tlsClient.Post(
		ssesc.PostSseScRequestPath, []byte(outputJSON))

	if err != nil {
		cl.rnrChan.Close(req.CorrelationID)
		slog.Warn("SendRequest ->: error in sending request",
			"dThingID", req.ThingID,
			"name", req.Name,
			"correlationID", req.CorrelationID,
			"err", err.Error())
		return err
	}

	if code == http.StatusOK || (code > 200 && code < 300) {
		// hiveot sse always uses the SSE return channel for the response message.
		// While code 200 could in theory include the response message in the http
		// response, hiveot chooses to always pass the response via SSE.
		// the reply from the RNR channel is sent directly to the given replyTo handler.
		cl.rnrChan.WaitWithCallback(req.CorrelationID, cl.GetTimeout(), replyTo)
	} else {
		// something went wrong and no response is expected, close the channel
		cl.rnrChan.Close(req.CorrelationID)
		// is this really unexpected?
		slog.Warn("SendRequest: unexpected result code", "code", code)

		// error result, no response is expected so create one
		// use error details in the output data if provided
		resp := req.CreateResponse(nil, nil)
		httpProblemDetail := map[string]string{}
		if len(outputRaw) > 0 {
			err = jsoniter.Unmarshal(outputRaw, &httpProblemDetail)
			resp.Error = &msg.ErrorValue{
				Status: code,
				Title:  httpProblemDetail["title"],
				Detail: httpProblemDetail["detail"],
			}
		} else {
			resp.Error = &msg.ErrorValue{
				Status: code,
				Title:  "request failed",
			}
		}
		_ = replyTo(resp)
	}
	return err
}

// SendResponse [Device] posts a response using the hiveot protocol.
//
// Used by devices when using reverse connection to a server.
func (cl *SseScClientImpl) SendResponse(resp *msg.ResponseMessage) error {

	// Send as text, not binary, to avoid unmarshalling problems
	outputJSON, _ := jsoniter.MarshalToString(resp)
	_, _, err := cl.tlsClient.Post(
		ssesc.PostSseScResponsePath, []byte(outputJSON))
	return err
}

// stop closes the connection
func (cl *SseScClientImpl) Stop() {
	cl.Close()
}

// StartSseScClientImpl starts a new instance of the hiveot http/sse-sc protocol binding client.
// This uses TD forms to perform operations.
// Set authentication and call connect before use.
//
// For testing, or very slow networks, use SetTimeout to increase the wait time.
//
//	sseURL full connection URL of Hiveot SSE server and path
//	rootCAs are CA certificates to validate the server certificate. nil for system CAs.
//	ch is the connect/disconnect callback
func StartSseScClientImpl(sseURL string, rootCAs *x509.CertPool) *SseScClientImpl {

	urlParts, err := url.Parse(sseURL)
	if err != nil {
		slog.Error("Invalid URL")
		return nil
	}
	hostPort := urlParts.Host
	ssePath := urlParts.Path
	// use SetTimeout to change the default
	timeout := msg.DefaultRnRTimeout
	tlsClient := tls_client.NewTLSClient(hostPort, rootCAs)

	thingID := ssesc.SseScClientCellType + shortid.MustGenerate()
	cl := &SseScClientImpl{
		TransportClientBase: transport.NewTransportClientBase(thingID, rootCAs, timeout),
		encoder:             transport.NewRRNJsonEncoder(),
		rnrChan:             msg.NewRnRChan(),
		ssePath:             ssePath,
		tlsClient:           tlsClient,
	}
	var _ api.IHiveCell = cl        // interface check
	var _ api.ITransportClient = cl // interface check
	return cl
}

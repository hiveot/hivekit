package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/modules/transports/httpclient"
	"github.com/hiveot/hivekit/go/modules/transports/ssesc"
	jsoniter "github.com/json-iterator/go"
	"github.com/teris-io/shortid"
	gosse "github.com/tmaxmax/go-sse"
)

// HiveotSseClient is the http client for connecting a WoT client to a http
// server using the HiveOT http and sse sub-protocol.
//
// This implements the IConnection and IHiveModule interfaces so it can be used as
// a regular client and as a sink for other modules.
//
// This can be used by both consumers and agents.
// This is intended to be used together with an SSE return channel.
//
// The Forms needed to invoke an operations are obtained using the 'getForm'
// callback, which can be tied to a store of TD documents. The form contains the
// hiveot RequestMessage and ResponseMessage endpoints. If no form is available
// then use the default hiveot endpoints that are defined with this protocol binding.
type HiveotSseClient struct {
	modules.HiveModuleBase

	connectHandler transports.ConnectionHandler

	isConnected atomic.Bool

	lastError atomic.Pointer[error]

	// encode/decode the request/response to the SSE messaging protocol used
	encoder transports.IMessageEncoder

	// sse variables access
	mux sync.RWMutex

	// notificationSink is the sink for forwarding notification messages to
	// this is the upstream consumer.
	notificationSink msg.NotificationHandler

	// requestSink is the sink for forwarding requests messages to
	requestSink msg.RequestHandler

	// the request & response channel handler to match requests and responses.
	// This is used in SendRequest to wait for the response received via SSE and pass it
	// to the replyTo callbacks.
	rnrChan *msg.RnRChan

	// the sse path for the connection
	ssePath string

	sseRetryOnDisconnect atomic.Bool

	// handler for closing the sse connection
	sseCancelFn context.CancelFunc

	// Destination for request/responses received from the server.
	// notifications received from the sink are sent to the server.
	// sink modules.IHiveModule

	// Timeout for http requests and SSE connect
	timeout time.Duration

	// http2 client for posting messages
	tlsClient transports.ITLSClient
}

// Authenticate the client connection with the server
// This determine which auth schema the TD describes, obtains the credentials
// and injects the authentication credentials according to the TDI schema.
// This returns an error if the schema isn't supported or is not compatible.
func (cl *HiveotSseClient) Authenticate(tdDoc *td.TD,
	getCredentials transports.GetCredentials) error {

	// HiveOT SSE-SC only uses bearer token
	clientID, secret, schemeName, err := getCredentials(tdDoc.ID)
	secScheme, err := tdDoc.GetSecurityScheme()

	if schemeName != secScheme.Scheme && schemeName != "" && schemeName != td.SecSchemeAuto {
		err = fmt.Errorf("Security scheme doesn't match credentials TD scheme='%s', credentials scheme='%s'", secScheme.Scheme, schemeName)
	} else if secScheme.Scheme == td.SecSchemeDigest {
		// err = cl.ConnectWithDigest(clientID, secret)
		err = fmt.Errorf("Digest authentication is not yet supported. Use bearer token instead")
	} else if secScheme.Scheme == td.SecSchemeBearer || secScheme.Scheme == td.SecSchemeAuto {
		err = cl.ConnectWithToken(clientID, secret)
	} else {
		err = fmt.Errorf("Unexpected security scheme '%s'", secScheme.Scheme)
	}
	return err
}

// ConnectSSE establishes the sse connection using the given bearer token
// cl.handleSseEvent will set 'connected' status when the first ping event is
// received from the server. (go-sse doesn't have a connected callback)
func (cl *HiveotSseClient) ConnectSSE(token string) (err error) {
	if cl.ssePath == "" {
		return fmt.Errorf("connectSSE: Missing SSE path")
	}
	// establish the SSE connection for the return channel
	//sseURL := fmt.Sprintf("https://%s%s", cc.hostPort, cc.ssePath)

	cl.sseCancelFn, err = ConnectSSE(
		// use the same http client for both http requests and sse connection
		cl.tlsClient,
		cl.ssePath,
		cl.handleSSEConnect,
		cl.handleSseEvent,
		cl.timeout)

	return err
}

// Connect authenticating using a client certificate
func (cl *HiveotSseClient) ConnectWithClientCert(clientCert *tls.Certificate) (err error) {
	return cl.tlsClient.ConnectWithClientCert(clientCert)
}

// ConnectWithToken sets the clientID and bearer token to use with requests and
//
//	establishes an SSE connection.
//
// If a connection exists it is closed first.
func (cl *HiveotSseClient) ConnectWithToken(clientID, token string) error {

	// ensure disconnected (note that this resets retryOnDisconnect)
	cl.Close()

	err := cl.tlsClient.ConnectWithToken(clientID, token)
	if err != nil {
		return err
	}
	// connectSSE will set 'isConnected' on success
	err = cl.ConnectSSE(token)
	if err != nil {
		cl.SetConnected(false)
		return err
	}
	return err
}

// Close the connection with the server
func (cl *HiveotSseClient) Close() {
	slog.Debug("HiveotSseClient.Disconnect",
		slog.String("clientID", cl.tlsClient.GetClientID()),
	)
	cl.mux.Lock()
	cb := cl.sseCancelFn
	cl.sseCancelFn = nil
	cl.mux.Unlock()

	// the connection status will update, if changed, through the sse callback
	if cb != nil {
		cb()
	}

	cl.mux.Lock()
	defer cl.mux.Unlock()
	if cl.IsConnected() {
		cl.tlsClient.Close()
	}
}

func (cl *HiveotSseClient) GetClientID() string {
	return cl.tlsClient.GetClientID()
}

// GetConnectionInfo returns the client's connection details
func (cl *HiveotSseClient) GetConnectionID() string {
	return cl.tlsClient.GetConnectionID()
}

// Provide the native http client used by this client
func (cl *HiveotSseClient) GetHttpClient() *http.Client {
	cl.mux.RLock()
	defer cl.mux.RUnlock()
	return cl.tlsClient.GetHttpClient()
}

func (cl *HiveotSseClient) GetTM() string {
	return ""
}

// HandleNotification receives an incoming notification from a producer
// and sends it to the server.
func (m *HiveotSseClient) HandleNotification(notif *msg.NotificationMessage) {

	m.SendNotification(notif)
}

// clients send requests to the server
func (cl *HiveotSseClient) HandleRequest(request *msg.RequestMessage, replyTo msg.ResponseHandler) error {
	err := cl.SendRequest(request, replyTo)
	return err
}

// handler when the SSE connection is established or fails.
// This invokes the connectHandler callback if provided.
func (cl *HiveotSseClient) handleSSEConnect(connected bool, err error) {
	errMsg := ""
	clientID := cl.GetClientID()
	cid := cl.tlsClient.GetConnectionID()

	// if the context is cancelled this is not an error
	if err != nil {
		errMsg = err.Error()
	}
	slog.Info("handleSSEConnect",
		slog.String("clientID", clientID),
		slog.String("connectionID", cid),
		slog.Bool("connected", connected),
		slog.String("err", errMsg))

	var connectionChanged bool = false
	if cl.IsConnected() != connected {
		connectionChanged = true
	}
	cl.SetConnected(connected)
	if err != nil {
		cl.mux.Lock()
		cl.lastError.Store(&err)
		cl.mux.Unlock()
	}

	// Note: this callback can send notifications to the client,
	// so prevent deadlock by running in the background.
	// (caught by readhistory failing for unknown reason)
	if connectionChanged && cl.connectHandler != nil {
		go func() {
			cl.connectHandler(connected, cl, err)
		}()
	}
}

// handleSSEEvent processes the push-event received from the server.
// This splits the message into notification, response and request
// requests have an operation and correlationID
// responses have no operations and a correlationID
// notifications have an operations and no correlationID
func (cl *HiveotSseClient) handleSseEvent(event gosse.Event) {
	clientID := cl.tlsClient.GetClientID()

	// no further processing of a ping needed
	if event.Type == ssesc.SSEPingEvent {
		return
	}

	// Use the hiveot message envelopes for request, response and notification
	switch event.Type {
	case msg.MessageTypeNotification:
		notif, err := cl.encoder.DecodeNotification([]byte(event.Data))
		if err != nil {
			return
		}
		// if cl.requestSink == nil {
		// 	slog.Error("HandleSseEvent: no sink set. Notification is dropped.",
		// 		"clientID", clientID,
		// 		"operation", notif.Operation,
		// 		"name", notif.Name,
		// 	)
		// } else
		if cl.notificationSink != nil {
			// notifications received from the server are passed to the registered handler
			go func() {
				cl.notificationSink(notif)
			}()
		} else {
			// notifications are only received when subscribed so someone forgot to
			// set a handler.
			slog.Error("handleSseEvent: Received notification but no handler is set")
		}
	case msg.MessageTypeRequest:
		var err error
		req, err := cl.encoder.DecodeRequest([]byte(event.Data))
		if err != nil {
			return
		}
		if cl.requestSink == nil {
			err = fmt.Errorf("handleSseEvent: no requestSink set. Request is dropped.")
			slog.Error("handleSseEvent: no sink set. Request is dropped.",
				"clientID", clientID,
				"operation", req.Operation,
				"name", req.Name,
				"senderID", req.SenderID,
			)
		} else {
			err = cl.requestSink(req, func(resp *msg.ResponseMessage) error {
				// return the response to the caller
				err2 := cl.SendResponse(resp)
				return err2
			})
			// an error means the request could not be handled
		}
		// responses are optional
		if err != nil {
			resp := req.CreateErrorResponse(err)
			_ = cl.SendResponse(resp)
		}

	case msg.MessageTypeResponse:
		resp, err := cl.encoder.DecodeResponse([]byte(event.Data))
		if err != nil {
			slog.Info("handleSseEvent: Received SSE Event but decoder returns nil", "data", string(event.Data))
			return
		}

		// consumer receives a response
		// this will be 'handled' if it was waiting on its rnr channel
		handled := cl.rnrChan.HandleResponse(resp, cl.timeout)

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
		// TBD: maybe this should just always fail?
		if cl.notificationSink == nil {
			slog.Error("handleSseEvent, received unexpected message",
				"messageType", event.Type)
			return
		}

		// all other events are intended for other use-cases such as the UI,
		// and can have a formats of event/{dThingID}/{name}
		// Attempt to deliver this for compatibility with other protocols (such has hiveoview test client)
		senderID := "" // unknown
		notif := msg.NewNotificationMessage(
			senderID, msg.AffordanceType(event.Type), "", "", event.Data)

		// don't block the receiver flow
		go func() {
			cl.notificationSink(notif)
		}()
	}
}

// IsConnected return whether the return channel is connection, eg can receive data
func (cl *HiveotSseClient) IsConnected() bool {
	return cl.isConnected.Load()
}

// SendNotification Agent posts a notification using the hiveot http/sse protocol.
//
// This posts the JSON-encoded NotificationMessage on the well-known hiveot notification path.
// In WoT Agents are typically a server, not a client, so this is intended for
// agents that use connection-reversal.
// Forms are not needed.
func (cl *HiveotSseClient) SendNotification(msg *msg.NotificationMessage) {
	// Send as text, not binary, to avoid unmarshalling problems
	outputJSON, _ := jsoniter.MarshalToString(msg)
	_, _, err := cl.tlsClient.Post(
		ssesc.PostSseScNotificationPath, []byte(outputJSON))

	if err != nil {
		slog.Warn("SendNotification failed",
			"clientID", cl.tlsClient.GetClientID(),
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
func (cl *HiveotSseClient) SendRequest(
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
		cl.rnrChan.WaitWithCallback(req.CorrelationID, cl.timeout, replyTo)
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

// SendResponse [Agent] posts a response using the hiveot protocol.
//
// Use by agent when using reverse connection to a server.
//
// This posts the JSON serialized ResponseMessage on the well-known hiveot-sse response href.
// Forms are not needed.
// In WoT Agents are typically a server, not a client, so this is intended for
// agents that use connection-reversal.
func (cl *HiveotSseClient) SendResponse(resp *msg.ResponseMessage) error {

	// Send as text, not binary, to avoid unmarshalling problems
	outputJSON, _ := jsoniter.MarshalToString(resp)
	_, _, err := cl.tlsClient.Post(
		ssesc.PostSseScResponsePath, []byte(outputJSON))
	return err
}

// SetBearerToken sets the authentication bearer token to authenticate http requests.
// func (cl *HiveotSseClient) SetBearerToken(token string) error {
// 	cl.mux.Lock()
// 	cl.bearerToken = token
// 	cl.mux.Unlock()
// 	return nil
// }

// SetConnected sets the sub-protocol connection status
func (cl *HiveotSseClient) SetConnected(isConnected bool) {
	cl.isConnected.Store(isConnected)
}

// SetNotificationSink sets the consumer handler for the notifications received
// from the server.
func (cl *HiveotSseClient) SetNotificationSink(sink msg.NotificationHandler) {
	cl.mux.Lock()
	cl.notificationSink = sink
	cl.mux.Unlock()
}

// SetRequestSink sets the agent module that handles requests received from the server.
func (cl *HiveotSseClient) SetRequestSink(sink msg.RequestHandler) {
	cl.mux.Lock()
	cl.requestSink = sink
	cl.mux.Unlock()
}

// SetTimeout sets the messaging timeout
func (cl *HiveotSseClient) SetTimeout(timeout time.Duration) {
	cl.timeout = timeout
	cl.tlsClient.SetTimeout(timeout)
}

// start doesn't do anything. Use ConnectWith... to connect.
// TBD: maybe this should connect using config?
func (cl *HiveotSseClient) Start() error {
	return nil
}

// stop closes the connection
func (cl *HiveotSseClient) Stop() {
	cl.Close()
}

// NewHiveotSseClient creates a new instance of the hiveot http/sse-sc protocol binding client.
// This uses TD forms to perform operations.
//
// For testing, or very slow networks, use SetTimeout to increase the wait time.
//
//	sseURL full connection URL of Hiveot SSE server and path
//	caCert is the CA certificate to validate the server certificate
//	ch is the connect/disconnect callback
func NewHiveotSseClient(sseURL string, caCert *x509.Certificate,
	ch transports.ConnectionHandler) *HiveotSseClient {

	urlParts, err := url.Parse(sseURL)
	if err != nil {
		slog.Error("Invalid URL")
		return nil
	}
	hostPort := urlParts.Host
	ssePath := urlParts.Path
	// use SetTimeout to change the default
	timeout := msg.DefaultRnRTimeout
	tlsClient := httpclient.NewHttpClient(hostPort, caCert, timeout)

	cl := &HiveotSseClient{
		connectHandler: ch,
		encoder:        transports.NewRRNJsonEncoder(),
		rnrChan:        msg.NewRnRChan(),
		ssePath:        ssePath,
		tlsClient:      tlsClient,
		timeout:        timeout,
	}
	var _ modules.IHiveModule = cl         // interface check
	var _ transports.ITransportClient = cl // interface check
	return cl
}

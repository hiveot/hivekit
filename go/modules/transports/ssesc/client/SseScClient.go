package ssescclient

import (
	"context"
	"crypto/x509"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/modules/transports/direct"
	tlsclient "github.com/hiveot/hivekit/go/modules/transports/httpserver/client"
	"github.com/hiveot/hivekit/go/modules/transports/ssesc"
	"github.com/hiveot/hivekit/go/msg"
	jsoniter "github.com/json-iterator/go"
	"github.com/teris-io/shortid"
)

// SseScClient is the http client for connecting a WoT client to a http
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
type SseScClient struct {
	connectHandler transports.ConnectionHandler

	isConnected atomic.Bool

	lastError atomic.Pointer[error]

	// convert the request/response to the SSE messaging protocol used
	msgConverter transports.IMessageConverter

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
	tlsClient transports.ITlsClient
}

// ConnectWithToken sets the clientID and bearer token to use with requests and
//
//	establishes an SSE connection.
//
// If a connection exists it is closed first.
func (cl *SseScClient) ConnectWithToken(clientID, token string, ch transports.ConnectionHandler) error {

	// ensure disconnected (note that this resets retryOnDisconnect)
	cl.Close()
	cl.connectHandler = ch

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
func (cl *SseScClient) Close() {
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

func (cl *SseScClient) GetClientID() string {
	return cl.tlsClient.GetClientID()
}

// GetConnectionInfo returns the client's connection details
func (cl *SseScClient) GetConnectionID() string {
	return cl.tlsClient.GetConnectionID()
}

// Provide the native http client used by this client
func (cl *SseScClient) GetHttpClient() *http.Client {
	cl.mux.RLock()
	defer cl.mux.RUnlock()
	return cl.tlsClient.GetHttpClient()
}
func (cl *SseScClient) GetModuleID() string {
	return cl.GetClientID()
}
func (cl *SseScClient) GetTM() string {
	return ""
}

// HandleNotification receives an incoming notification from a producer
// and sends it to the server.
func (m *SseScClient) HandleNotification(notif *msg.NotificationMessage) {

	m.SendNotification(notif)
}

// clients send requests to the server
func (cl *SseScClient) HandleRequest(request *msg.RequestMessage, replyTo msg.ResponseHandler) error {
	err := cl.SendRequest(request, replyTo)
	return err
}

// IsConnected return whether the return channel is connection, eg can receive data
func (cl *SseScClient) IsConnected() bool {
	return cl.isConnected.Load()
}

// SendNotification Agent posts a notification using the hiveot http/sse protocol.
//
// This posts the JSON-encoded NotificationMessage on the well-known hiveot notification path.
// In WoT Agents are typically a server, not a client, so this is intended for
// agents that use connection-reversal.
// Forms are not needed.
func (cl *SseScClient) SendNotification(msg *msg.NotificationMessage) {
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
func (cl *SseScClient) SendRequest(
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
		cl.rnrChan.WaitWithCallback(req.CorrelationID, replyTo)
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
func (cl *SseScClient) SendResponse(resp *msg.ResponseMessage) error {

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
func (cl *SseScClient) SetConnected(isConnected bool) {
	cl.isConnected.Store(isConnected)
}

// SetNotificationSink sets the consumer handler for the notifications received
// from the server.
func (cl *SseScClient) SetNotificationSink(sink msg.NotificationHandler) {
	cl.mux.Lock()
	cl.notificationSink = sink
	cl.mux.Unlock()
}

// SetRequestSink sets the agent module that handles requests received from the server.
func (cl *SseScClient) SetRequestSink(sink msg.RequestHandler) {
	cl.mux.Lock()
	cl.requestSink = sink
	cl.mux.Unlock()
}

// start doesn't do anything. Use ConnectWith... to connect.
// TBD: maybe this should connect using config?
func (cl *SseScClient) Start(yamlConfig string) error {
	return nil
}

// stop closes the connection
func (cl *SseScClient) Stop() {
	cl.Close()
}

// NewSseScClient creates a new instance of the http-basic protocol binding client.
// This uses TD forms to perform an operation.
//
// Users must use ConnectWithToken to authenticate and connect.
//
//	sseURL full connection URL of SSE server and path
//	caCert is the CA certificate to validate the server certificate
//	sink is the application module receiving notifications or in case of agents, requests.
//	timeout for waiting for response. 0 to use the default.
func NewSseScClient(
	sseURL string, caCert *x509.Certificate, timeout time.Duration) *SseScClient {

	urlParts, err := url.Parse(sseURL)
	if err != nil {
		slog.Error("Invalid URL")
		return nil
	}
	hostPort := urlParts.Host
	ssePath := urlParts.Path

	if timeout == 0 {
		timeout = transports.DefaultRpcTimeout
	}

	tlsClient := tlsclient.NewTLSClient(hostPort, nil, caCert, timeout)

	cl := &SseScClient{
		msgConverter: direct.NewPassthroughMessageConverter(),
		rnrChan:      msg.NewRnRChan(timeout),
		ssePath:      ssePath,
		tlsClient:    tlsClient,
		timeout:      timeout,
	}
	var _ transports.IConnection = cl // interface check
	var _ modules.IHiveModule = cl    // interface check
	return cl
}

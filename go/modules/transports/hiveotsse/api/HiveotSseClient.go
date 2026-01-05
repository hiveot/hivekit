package sseapi

import (
	"context"
	"crypto/x509"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hiveot/hivekit/go/lib/servers/httpbasic"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/modules/transports/hiveotsse"
	"github.com/hiveot/hivekit/go/modules/transports/httpserver/httpapi"
	"github.com/hiveot/hivekit/go/msg"
	jsoniter "github.com/json-iterator/go"
	"github.com/teris-io/shortid"
)

// HiveotSseClient is the http client for connecting a WoT client to a http
// server using the HiveOT http and sse sub-protocol.
//
// This based on the HttpBasic client and implements the IClientConnection interface.
//
// This can be used by both consumers and agents.
// This is intended to be used together with an SSE return channel.
//
// The Forms needed to invoke an operations are obtained using the 'getForm'
// callback, which can be tied to a store of TD documents. The form contains the
// hiveot RequestMessage and ResponseMessage endpoints. If no form is available
// then use the default hiveot endpoints that are defined with this protocol binding.
type HiveotSseClient struct {
	appConnectHandlerPtr atomic.Pointer[transports.ConnectionHandler]

	// authentication bearer token if authenticated
	bearerToken string

	// Connection information such as clientID, cid, address, protocol etc
	cinfo transports.ConnectionInfo

	// convert the request/response to the wss messaging protocol used
	msgConverter transports.IMessageConverter

	// the request & response channel handler
	// See also ConnectSSE where all responses are passed to this to support
	// replyTo callbacks.
	rnrChan *transports.RnRChan

	// the sse connection path
	ssePath              string
	sseRetryOnDisconnect atomic.Bool
	// handler for closing the sse connection
	sseCancelFn context.CancelFunc

	isConnected atomic.Bool

	// sse variables access
	mux sync.RWMutex

	// destination for notifications, requests and responses.
	// This is intended to be the application module the client connects to.
	sink modules.IHiveModule

	// http2 client for posting messages
	tlsClient *httpapi.TLSClient

	lastError atomic.Pointer[error]
}

// ConnectWithToken sets the bearer token to use with requests and establishes
// an SSE connection.
// If a connection exists it is closed first.
func (cl *HiveotSseClient) ConnectWithToken(token string) error {

	// ensure disconnected (note that this resets retryOnDisconnect)
	cl.Disconnect()

	err := cl.SetBearerToken(token)
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

// Disconnect from the server
func (cl *HiveotSseClient) Disconnect() {
	slog.Debug("HiveotSseClient.Disconnect",
		slog.String("clientID", cl.cinfo.ClientID),
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

// GetAppConnectHandler returns the application handler for connection status updates
func (cl *HiveotSseClient) GetAppConnectHandler() transports.ConnectionHandler {
	hPtr := cl.appConnectHandlerPtr.Load()
	return *hPtr
}

func (cl *HiveotSseClient) GetClientID() string {
	return cl.cinfo.ClientID
}

// GetConnectionInfo returns the client's connection details
func (cl *HiveotSseClient) GetConnectionInfo() transports.ConnectionInfo {
	return cl.cinfo
}
func (cl *HiveotSseClient) GetTlsClient() *http.Client {
	cl.mux.RLock()
	defer cl.mux.RUnlock()
	return cl.tlsClient.GetHttpClient()
}

// IsConnected return whether the return channel is connection, eg can receive data
func (cl *HiveotSseClient) IsConnected() bool {
	return cl.isConnected.Load()
}

// LoginWithPassword posts a login request to the TLS server using a login ID and
// password and obtain an auth token for use with SetBearerToken.
// This uses the http-basic login endpoint.
//
// FIXME: use a WoT standardized auth method
//
// If the connection fails then any existing connection is cancelled.
func (cl *HiveotSseClient) LoginWithPassword(password string) (newToken string, err error) {

	slog.Info("ConnectWithPassword",
		"clientID", cl.GetClientID(), "connectionID", cl.GetConnectionInfo().ConnectionID)

	// FIXME: figure out how a standard login method is used to obtain an auth token
	args := transports.UserLoginArgs{
		Login:    cl.GetClientID(),
		Password: password,
	}

	argsJSON, _ := jsoniter.Marshal(args)
	outputRaw, _, err := cl.tlsClient.Post(
		httpbasic.HttpPostLoginPath, []byte(argsJSON))

	if err == nil {
		err = jsoniter.Unmarshal(outputRaw, &newToken)
	}
	// store the bearer token further requests
	// when login fails this clears the existing token. Someone else
	// logging in cannot continue on a previously valid token.
	cl.mux.Lock()
	cl.bearerToken = newToken
	cl.mux.Unlock()
	//cl.BaseIsConnected.Store(true)
	if err != nil {
		slog.Warn("connectWithPassword failed: " + err.Error())
	}

	return newToken, err
}

// SendNotification Agent posts a notification using the hiveot http/sse protocol.
//
// This posts the JSON-encoded NotificationMessage on the well-known hiveot notification path.
// In WoT Agents are typically a server, not a client, so this is intended for
// agents that use connection-reversal.
// Forms are not needed.
//
// This returns an error if the notification could not be delivered to the server
func (cl *HiveotSseClient) SendNotification(msg *msg.NotificationMessage) error {
	// Send as text, not binary, to avoid unmarshalling problems
	outputJSON, _ := jsoniter.MarshalToString(msg)
	_, _, err := cl.tlsClient.Post(
		hiveotsse.PostHiveotSseNotificationPath, []byte(outputJSON))

	if err != nil {
		slog.Warn("SendNotification failed",
			"clientID", cl.cinfo.ClientID,
			"err", err.Error())
	}
	return err
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
			hiveotsse.PostHiveotSseRequestPath, []byte(outputJSON))
		_ = code
		_ = outputRaw

		return err
	}

	// A response handler is provided. Invoke replyTo when the response is received
	// via sse.
	rChan := cl.rnrChan.Open(req.CorrelationID)
	_ = rChan

	outputRaw, code, err := cl.tlsClient.Post(
		hiveotsse.PostHiveotSseRequestPath, []byte(outputJSON))

	if err != nil {
		slog.Warn("SendRequest ->: error in sending request",
			"dThingID", req.ThingID,
			"name", req.Name,
			"correlationID", req.CorrelationID,
			"err", err.Error())
		return err
	}

	if code == http.StatusOK || (code > 200 && code < 300) {
		// successful call
		cl.rnrChan.WaitWithCallback(req.CorrelationID, replyTo, 0)
	} else {
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
		hiveotsse.PostHiveotSseResponsePath, []byte(outputJSON))
	return err
}

// SetBearerToken sets the authentication bearer token to authenticate http requests.
func (cl *HiveotSseClient) SetBearerToken(token string) error {
	cl.mux.Lock()
	cl.bearerToken = token
	cl.mux.Unlock()
	return nil
}

// SetConnected sets the sub-protocol connection status
func (cl *HiveotSseClient) SetConnected(isConnected bool) {
	cl.isConnected.Store(isConnected)
}

// SetConnectHandler set the application handler for connection status updates
func (cl *HiveotSseClient) SetConnectHandler(cb transports.ConnectionHandler) {
	if cb == nil {
		cl.appConnectHandlerPtr.Store(nil)
	} else {
		cl.appConnectHandlerPtr.Store(&cb)
	}
}

// SetSink set the application module that handles async notifications, requests and responses
func (cl *HiveotSseClient) SetSink(sink modules.IHiveModule) {
	cl.mux.Lock()
	cl.sink = sink
	cl.mux.Unlock()
}

// NewHiveotSseClient creates a new instance of the http-basic protocol binding client.
// This uses TD forms to perform an operation.
//
//	sseURL of the http and sse server to connect to, including the schema
//	clientID to identify as. Must match the auth token
//	caCert of the server to validate the server or nil to not check the server cert
//	sink is the application module receiving notifications or in case of agents, requests.
//	timeout for waiting for response. 0 to use the default.
func NewHiveotSseClient(
	sseURL string, clientID string, caCert *x509.Certificate,
	sink modules.IHiveModule, timeout time.Duration) *HiveotSseClient {

	urlParts, err := url.Parse(sseURL)
	if err != nil {
		slog.Error("Invalid URL")
		return nil
	}
	hostPort := urlParts.Host
	ssePath := urlParts.Path
	tlsClient := httpapi.NewTLSClient(hostPort, nil, caCert, timeout)

	cl := HiveotSseClient{
		cinfo: transports.ConnectionInfo{
			CaCert:       caCert,
			ClientID:     clientID,
			ConnectionID: "sse-" + shortid.MustGenerate(),
			ConnectURL:   sseURL,
			// ProtocolType: msg.ProtocolTypeHiveotSSE,
			Timeout: timeout,
		},
		ssePath: ssePath,
		// hostPort:  hostPort,
		sink:      sink,
		tlsClient: tlsClient,
	}
	return &cl
}

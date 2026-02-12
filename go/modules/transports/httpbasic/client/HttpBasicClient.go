package httpbasicclient

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/modules/transports/httpbasic"
	tlsclient "github.com/hiveot/hivekit/go/modules/transports/httpserver/client"
	"github.com/hiveot/hivekit/go/msg"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hivekit/go/wot"
	"github.com/hiveot/hivekit/go/wot/td"
	jsoniter "github.com/json-iterator/go"
)

// HttpBasicClient is the RRN messaging client for connecting a WoT client to a WoT server
// over http/2 using the WoT http-basic protocol profile.
// This implements the IClientConnection interface.
//
// This can be used alone or with the hiveotsseclient which provides an SSE return channel.
// This provides authentication methods.
//
// The Forms needed to invoke an operations are obtained using the 'getForm'
// callback, which can be tied to a store of TD documents. The form contains the
// hiveot RequestMessage and ResponseMessage endpoints. If no form is available
// then use the default hiveot endpoints that are defined with this protocol binding.
type HttpBasicClient struct {

	// handler for requests send by clients
	connectHandler transports.ConnectionHandler

	//clientID string
	// Connection information such as clientID, cid, address, protocol etc
	// cinfo transports.ConnectionInfo

	// getForm obtains the form for sending a request or notification
	// if nil, then the hiveot protocol envelope and URL are used as fallback
	getForm transports.GetFormHandler

	isConnected atomic.Bool

	// protected operations
	mux sync.RWMutex

	// destination for notifications, requests and responses.
	// This is intended to be the application module the client connects to.
	sink modules.IHiveModule

	// timeout for use with SendRequest
	timeout time.Duration

	// http2 client for posting messages
	tlsClient transports.ITlsClient
}

// Set the clientID and authentication bearer token.
// This performs a standard /ping health check that the hiveot http server supports.
func (cl *HttpBasicClient) ConnectWithToken(
	clientID string, token string, ch transports.ConnectionHandler) error {

	cl.connectHandler = ch
	err := cl.tlsClient.ConnectWithToken(clientID, token)
	if err == nil {
		var status int
		// TBD: should ping always work?
		status, err = cl.tlsClient.Ping()
		if status == http.StatusOK {
			cl.isConnected.Store(true)
		} else {
			cl.isConnected.Store(false)
		}
		// notify if interested
		if ch != nil {
			ch(cl.isConnected.Load(), cl, nil)
		}
	}
	return err
}

// Close disconnects from the server
func (cl *HttpBasicClient) Close() {

	cl.mux.Lock()
	defer cl.mux.Unlock()
	if cl.isConnected.Load() {
		cl.tlsClient.Close()
		cl.isConnected.Store(false)
		if cl.connectHandler != nil {
			cl.connectHandler(false, cl, nil)
		}
	}
}

// GetAppConnectHandler returns the application handler for connection status updates
// func (cl *HttpBasicClient) GetAppConnectHandler() transports.ConnectionHandler {
// 	hPtr := cl.appConnectHandlerPtr.Load()
// 	return *hPtr
// }

func (cl *HttpBasicClient) GetClientID() string {
	return cl.tlsClient.GetClientID()
}
func (cl *HttpBasicClient) GetConnectionID() string {
	return cl.tlsClient.GetConnectionID()
}

// GetDefaultForm return the default http form for the operation
// This simply returns nil for anything else than login, logout, ping or refresh.
func (cl *HttpBasicClient) GetDefaultForm(op, thingID, name string) (f *td.Form) {
	// login has its own URL as it is unauthenticated
	if op == wot.HTOpPing {
		href := transports.DefaultPingPath
		nf := td.NewForm(op, href)
		nf.SetMethodName(http.MethodGet)
		f = &nf
	}
	// everything else has no default form, so falls back to hiveot protocol endpoints
	return f
}
func (cl *HttpBasicClient) GetModuleID() string {
	return cl.GetClientID()
}

// Return the TLS client used by this connection
func (cl *HttpBasicClient) GetTlsClient() transports.ITlsClient {
	cl.mux.RLock()
	defer cl.mux.RUnlock()
	return cl.tlsClient
}
func (cl *HttpBasicClient) GetTM() string {
	return ""
}

// HandleNotification receives an incoming notification from a producer
// and sends it to the server.
func (m *HttpBasicClient) HandleNotification(notif *msg.NotificationMessage) {
	// Can't use HiveModuleBase.HandleNotification as it forwards the notification
	// to the registered notification sink.
	m.SendNotification(notif)
}

// clients send requests to the server
func (cl *HttpBasicClient) HandleRequest(request *msg.RequestMessage, replyTo msg.ResponseHandler) error {
	err := cl.SendRequest(request, replyTo)
	return err
}

// IsConnected return whether the return channel is connection, eg can receive data
func (cl *HttpBasicClient) IsConnected() bool {
	return cl.isConnected.Load()
}

// SendNotification is not supported in http-basic
func (cl *HttpBasicClient) SendNotification(msg *msg.NotificationMessage) {
	slog.Error("HttpBasic doesn't support sending notifications")
}

// SendRequest sends a request over http message using the form based path and passes
// the result as a response to the replyTo handler.
//
// This locates the form for the operation using 'getForm' and uses the result
// to determine the URL to publish the request to and if the hiveot RequestMessage
// envelope is used.
//
// If no form is found then fall back to the hiveot default paths.
// The request input, if any, is json encoded into the body of the request.
// This does not use a RequestMessage envelope to remain http-basic compatible.
//
// The response follows the http-basic specification:
// * code 200: completed; body is output
// * code 201: pending; body is http action status message
// * code 40x: failed ; body is error payload, if present
// * code 50x: failed ; body is error payload, if present
//
// This returns nil if the request was successfully sent or an error if the send failed.
// If the response has an error or is missing this invokes the replyTo with an error response and returns nil.
func (cl *HttpBasicClient) SendRequest(
	req *msg.RequestMessage, replyTo msg.ResponseHandler) error {

	var inputJSON string
	var method string
	var href string
	var thingID = req.ThingID
	var name = req.Name

	if req.Operation == "" && req.CorrelationID == "" {
		err := fmt.Errorf("SendMessage: missing both operation and correlationID")
		slog.Error(err.Error())
		return err
	}

	// the getForm callback provides the method and URL to invoke for this operation.
	// use the hiveot fallback if not available
	// If a form is provided and it doesn't use the hiveot subprotocol then fall
	// back to invoking using http basic using the form href.
	f := cl.getForm(req.Operation, req.ThingID, req.Name)
	if f != nil {
		method, _ = f.GetMethodName()
		href = f.GetHRef()
	}

	if f == nil {
		// fall back to the 'well known' hiveot request URL using uri variables
		// eg: /things/{operation}/{thingID}/{name} or /hiveot/request
		method = http.MethodPost
		href = httpbasic.HttpBasicAffordanceOperationPath
		inputJSON, _ = jsoniter.MarshalToString(req.Input)
	}

	// Inject URI variables for hrefs that use them:
	//  use + as wildcard for thingID to avoid a 404
	//  while not recommended, it is allowed to subscribe/observe all things
	if thingID == "" {
		thingID = "+"
	}
	//  use + as wildcard for affordance name to avoid a 404
	//  this should not happen very often but it is allowed
	if name == "" {
		name = "+"
	}
	// substitute URI variables in the path, if any.
	// intended for use with http-basic forms.
	vars := map[string]string{
		transports.ThingIDURIVar:   thingID,
		transports.NameURIVar:      name,
		transports.OperationURIVar: req.Operation}
	reqPath := utils.Substitute(href, vars)
	contentType := "application/JSON"

	// send the request
	ctx, cancelFn := context.WithTimeout(context.Background(), cl.timeout)
	outputRaw, code, _, err := cl.tlsClient.Send(ctx,
		method, reqPath, nil, []byte(inputJSON), contentType)
	cancelFn()

	// 1. error response
	if err != nil {
		return err
	}
	// follow the HTTP Basic specification
	if code == http.StatusOK {
		resp := req.CreateResponse(nil, nil)
		// unmarshal output. This is the json encoded output
		if len(outputRaw) == 0 {
			// nothing to unmarshal
		} else {
			err = jsoniter.UnmarshalFromString(string(outputRaw), &resp.Output)
		}
		if err != nil {
			resp.Error = msg.ErrorValueFromError(err)
			resp.Error.Status = 500 // decode error
		}

		// pass a direct response to the application handler
		err = replyTo(resp)
		// h := cc.GetAppResponseHandler()
		// go func() {
		// 	_ = h(resp)
		// }()
	} else if code > 200 && code < 300 {
		// httpbasic servers/things might respond with 201 for pending as per spec
		// this is a response message.
		var resp *msg.ResponseMessage
		if len(outputRaw) == 0 {
			// no response yet. do not send process a notification
		} else {
			// standard http response payload
			var tmp any
			err = jsoniter.Unmarshal(outputRaw, &tmp)
			resp = req.CreateResponse(tmp, err)
		}

		// pass a direct response to the application handler
		if resp != nil {
			_ = replyTo(resp)
			// h := cc.GetAppResponseHandler()
			// go func() {
			// 	_ = h(resp)
			// }()
		}
	} else {
		// unknown response, create an error response
		resp := req.CreateResponse(nil, nil)
		// unmarshal output. This is either the json encoded output or the ResponseMessage envelope
		if outputRaw == nil {
			// nothing to unmarshal
		} else {
			err = jsoniter.UnmarshalFromString(string(outputRaw), &resp.Output)
		}
		httpProblemDetail := map[string]string{}
		if len(outputRaw) > 0 {
			err = jsoniter.Unmarshal(outputRaw, &httpProblemDetail)
			statusCode := utils.DecodeAsInt(httpProblemDetail["status"])
			resp.Error = &msg.ErrorValue{
				Status: statusCode,
				Title:  httpProblemDetail["title"],
				Detail: httpProblemDetail["detail"],
			}
		} else if err != nil {
			resp.Error = msg.ErrorValueFromError(err)
		} else {
			resp.Error = &msg.ErrorValue{
				Status: code,
				Title:  "request failed",
			}

		}

		// pass a direct response to the application handler
		replyTo(resp)
		// h := cc.GetAppResponseHandler()
		// go func() {
		// 	_ = h(resp)
		// }()
	}
	return err
}

// SendResponse is not supported in http-basic
func (cl *HttpBasicClient) SendResponse(resp *msg.ResponseMessage) error {
	return errors.New("HttpBasic doesn't support sending async responses")
}

// Does reports an error as http clients dont receive notifications
func (cl *HttpBasicClient) SetNotificationSink(cb msg.NotificationHandler) {
	slog.Warn("SetNotificationSink: HttpBasicClients dont handle notifications",
		"clientID", cl.GetClientID())
}

// SetRequestSink set sink that handles requests
// Since http-basic is a uni-directional transport client, requests are send to the server
// instead of passing it to this sink. Therefore this logs an error.
func (cl *HttpBasicClient) SetRequestSink(sink msg.RequestHandler) {
	slog.Warn("SetRequestSink. HttpBasicClient cannot be a request sink.")
}

func (cl *HttpBasicClient) SetTimeout(timeout time.Duration) {
	cl.timeout = timeout
	cl.tlsClient.SetTimeout(timeout)
}

// start doesn't do anything. Use ConnectWith... to connect.
// TBD: maybe this should connect using config?
func (cl *HttpBasicClient) Start(yamlConfig string) error {
	return nil
}

// stop closes the connection
func (cl *HttpBasicClient) Stop() {
	cl.Close()
}

// NewHttpBasicClient creates a new instance of the WoT compatible http-basic
// protocol binding client.
//
// Users must use ConnectWithToken to authenticate and connect.
//
// This uses TD forms to perform an operation.
//
//	baseURL of the http server. Used as the base for all further requests.
//	clientID to identify as. Must match the authentication information.
//	caCert of the server to validate the server or nil to not check the server cert
//	getForm is the handler for return a form for invoking an operation. nil for default
//	sink is the application module receiving notifications or in case of agents, requests.
//	timeout for waiting for response. 0 to use the default.
func NewHttpBasicClient(
	baseURL string, caCert *x509.Certificate, getForm transports.GetFormHandler) *HttpBasicClient {

	timeout := tlsclient.DefaultClientTimeout
	urlParts, err := url.Parse(baseURL)
	if err != nil {
		slog.Error("Invalid URL")
		return nil
	}
	hostPort := urlParts.Host

	tlsClient := tlsclient.NewTLSClient(hostPort, nil, caCert, timeout)

	cl := &HttpBasicClient{
		getForm:   getForm,
		timeout:   timeout,
		tlsClient: tlsClient,
	}
	if cl.getForm == nil {
		cl.getForm = cl.GetDefaultForm
	}
	var _ transports.IConnection = cl // interface check
	var _ modules.IHiveModule = cl    // interface check
	return cl
}

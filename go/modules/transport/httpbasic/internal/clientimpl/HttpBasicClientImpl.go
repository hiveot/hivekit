package clientimpl

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/transport/httpbasic"
	"github.com/hiveot/hivekit/go/modules/transport/tlsclient"
	tls_client "github.com/hiveot/hivekit/go/modules/transport/tlsclient/client"
	"github.com/hiveot/hivekit/go/utils"
	jsoniter "github.com/json-iterator/go"
	"github.com/teris-io/shortid"
)

// HttpBasicClientImpl is the RRN messaging client for connecting a WoT client to a WoT server
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
type HttpBasicClientImpl struct {
	*modules.HiveModuleBase

	// auth token when connecting with token
	bearerToken string

	// current connection status
	connectStatus api.ConnectionStatus
	// callback when connection changes
	connectHandler func(newStatus api.ConnectionStatus, c api.ITransportClient)

	// getForm obtains the form for sending a request or notification
	// if nil, then the hiveot protocol envelope and URL are used as fallback
	getForm api.GetFormHandler

	// protected operations
	mux sync.RWMutex

	// destination for notifications, requests and responses.
	// This is intended to be the application module the client connects to.
	sink api.IHiveModule

	// timeout for use with SendRequest
	timeout time.Duration

	// http2 client for posting messages
	tlsClient tlsclient.ITLSClient
}

// update the connection status and publish an notification if it differs from the last status
// a 'lost' status is ignored if the current status is set to closed as it was intentional.
func (cl *HttpBasicClientImpl) _setConnectionStatus(
	newStatus api.ConnectionStatus, err error) {

	cl.mux.RLock()
	oldStatus := cl.connectStatus
	cl.mux.RUnlock()

	if newStatus == oldStatus {
		return
	} else if oldStatus == api.StatusClosed && newStatus == api.StatusLost {
		return
	}
	cl.mux.Lock()
	cl.connectStatus = newStatus
	ch := cl.connectHandler
	cl.mux.Unlock()

	// notify upstream of status change
	moduleID := cl.GetThingID()
	evName := api.ClientConnectionStatusEvent
	notif := msg.NewNotificationMessage(
		moduleID, msg.AffordanceTypeEvent, moduleID, evName, newStatus)
	cl.ForwardNotification(notif)

	// invoke the callback after the notification so that the proper sequence is maintained
	// if the callback tries to reconnect.
	if ch != nil {
		ch(newStatus, cl)
	}
}

// Connect authenticating using a client certificate
func (cl *HttpBasicClientImpl) AuthenticateWithClientCert(clientCert *tls.Certificate) (err error) {
	return cl.tlsClient.AuthenticateWithClientCert(clientCert)
}

// Authenticate the client connection with the server
// This determine which auth schema the TD describes, obtains the credentials
// and injects the authentication credentials according to the TDI schema.
// This returns an error if the schema isn't supported or is not compatible.
func (cl *HttpBasicClientImpl) AuthenticateWithForm(
	tdDoc *td.TD, getCredentials api.GetCredentials) error {

	clientID, secret, schemeName, err := getCredentials(tdDoc.ID)
	secScheme, err := tdDoc.GetSecurityScheme()

	if schemeName != secScheme.Scheme && schemeName != "" && schemeName != td.SecSchemeAuto {
		err = fmt.Errorf("Security scheme doesn't match credentials TD scheme='%s', credentials scheme='%s'", secScheme.Scheme, schemeName)
	} else if secScheme.Scheme == td.SecSchemeDigest {
		// err = cl.ConnectWithDigest(clientID, secret)
		err = fmt.Errorf("Digest authentication is not yet supported. Use bearer token instead")
	} else if secScheme.Scheme == td.SecSchemeBearer || secScheme.Scheme == td.SecSchemeAuto {
		err = cl.AuthenticateWithToken(clientID, secret)
	} else {
		err = fmt.Errorf("Unexpected security scheme '%s'", secScheme.Scheme)
	}
	return err
}

// Set the clientID and authentication bearer token.
func (cl *HttpBasicClientImpl) AuthenticateWithToken(
	clientID string, token string) error {

	cl.bearerToken = token
	err := cl.tlsClient.AuthenticateWithToken(clientID, token)
	return err
}

// Close disconnects from the server
func (cl *HttpBasicClientImpl) Close() {

	// set status to closed first to avoid a reconnect
	cl._setConnectionStatus(api.StatusClosed, nil)

	cl.mux.Lock()
	defer cl.mux.Unlock()
	if cl.tlsClient != nil {
		cl.tlsClient.Close()
	}
}

// This performs a standard /ping health check that the hiveot http server supports.
func (cl *HttpBasicClientImpl) Connect() error {

	cl._setConnectionStatus(api.StatusConnecting, nil)
	statusCode, err := cl.tlsClient.Ping()
	if statusCode == http.StatusOK {
		cl._setConnectionStatus(api.StatusConnected, err)
	} else if statusCode == http.StatusUnauthorized {
		cl._setConnectionStatus(api.StatusRefused, err)
	} else {
		cl._setConnectionStatus(api.StatusLost, err)
	}
	return err
}

func (cl *HttpBasicClientImpl) GetClientID() string {
	return cl.tlsClient.GetClientID()
}
func (cl *HttpBasicClientImpl) GetConnectionID() string {
	return cl.tlsClient.GetConnectionID()
}

// // GetConnectionStatus returns the current connection status
func (cl *HttpBasicClientImpl) GetConnectionStatus() api.ConnectionStatus {
	cl.mux.RLock()
	defer cl.mux.RUnlock()
	stat := cl.connectStatus
	return stat
}

// GetDefaultForm return the default http form for the operation
// This simply returns nil for anything else than login, logout, ping or refresh.
func (cl *HttpBasicClientImpl) GetDefaultForm(op, thingID, name string) (f *td.Form, href string) {
	// login has its own URL as it is unauthenticated
	if op == td.HTOpPing {
		base := cl.tlsClient.GetHostPort()
		href = fmt.Sprintf("https://%s%s", base, api.DefaultPingPath)
		nf := td.NewForm(op, href)
		nf.SetMethodName(http.MethodGet)
		f = &nf
	}
	// everything else has no default form, so falls back to hiveot protocol endpoints
	return f, href
}

// Return the TLS client used by this connection
func (cl *HttpBasicClientImpl) GetTlsClient() tlsclient.ITLSClient {
	cl.mux.RLock()
	defer cl.mux.RUnlock()
	return cl.tlsClient
}
func (cl *HttpBasicClientImpl) GetTM() string {
	return ""
}

// HandleNotification receives an incoming notification from a producer
// and sends it to the server.
func (m *HttpBasicClientImpl) HandleNotification(notif *msg.NotificationMessage) {
	// Can't use HiveModuleBase.HandleNotification as it forwards the notification
	// to the registered notification sink.
	m.SendNotification(notif)
}

// clients send requests to the server
func (cl *HttpBasicClientImpl) HandleRequest(request *msg.RequestMessage, replyTo msg.ResponseHandler) error {
	err := cl.SendRequest(request, replyTo)
	return err
}

// SendNotification is not supported in http-basic
func (cl *HttpBasicClientImpl) SendNotification(msg *msg.NotificationMessage) {
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
func (cl *HttpBasicClientImpl) SendRequest(
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
	f, href := cl.getForm(req.Operation, req.ThingID, req.Name)
	if f != nil {
		method, _ = f.GetMethodName()
	}

	if f == nil {
		// fall back to the 'well known' hiveot request URL using uri variables
		// eg: /things/{op}/{id}/{name} or /hiveot/request
		method = http.MethodPost
		href = httpbasic.HttpBasicAffordanceOperationPath
		// substitute URI variables in the path, if any.
		// intended for use with http-basic forms.
		vars := map[string]string{
			td.UriVarThingID:   thingID,
			td.UriVarName:      name,
			td.UriVarOperation: req.Operation}
		href = utils.Substitute(href, vars)
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
	contentType := "application/JSON"

	// send the request
	ctx, cancelFn := context.WithTimeout(context.Background(), cl.timeout)
	outputRaw, code, _, err := cl.tlsClient.Send(ctx,
		method, href, nil, []byte(inputJSON), contentType)
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
func (cl *HttpBasicClientImpl) SendResponse(resp *msg.ResponseMessage) error {
	return errors.New("HttpBasic doesn't support sending async responses")
}

// Does reports an error as http clients dont receive notifications
func (cl *HttpBasicClientImpl) SetNotificationSink(sink api.IHiveModule, thingID ...string) {
	slog.Warn("SetNotificationSink: HttpBasicClients dont handle notifications",
		"clientID", cl.GetClientID())
}

// SetRequestSink set sink that handles requests
// Since http-basic is a uni-directional transport client, requests are send to the server
// instead of passing it to this sink. Therefore this logs an error.
func (cl *HttpBasicClientImpl) SetRequestSink(sink api.IHiveModule) {
	slog.Warn("SetRequestSink. HttpBasicClient cannot be a request sink.")
}

// SetConnectHandler sets the callback to invoke when the connection status changes
func (cl *HttpBasicClientImpl) SetConnectHandler(
	h func(newStatus api.ConnectionStatus, c api.ITransportClient)) {
	cl.mux.Lock()
	defer cl.mux.Unlock()
	cl.connectHandler = h
}

func (cl *HttpBasicClientImpl) SetTimeout(timeout time.Duration) {
	cl.timeout = timeout
	cl.tlsClient.SetTimeout(timeout)
}

// start doesn't do anything. Use ConnectWith... to connect.
// TBD: maybe this should connect using config?
func (cl *HttpBasicClientImpl) Start() error {
	return nil
}

// stop closes the connection
func (cl *HttpBasicClientImpl) Stop() {
	cl.Close()
}

// NewHttpBasicClientImpl creates a new instance of the WoT compatible http-basic
// protocol binding client.
//
// Users must use AuthenticateWithToken to authenticate and connect.
//
// This uses TD forms to perform an operation.
//
//	baseURL of the http server. Used as the base for all further requests.
//	caCert of the server to validate the server or nil to not check the server cert
//	getForm is the handler for return a form for invoking an operation. nil for default
//	ch optional callback with connection status changes
func NewHttpBasicClientImpl(
	baseURL string, caCert *x509.Certificate, getForm api.GetFormHandler) *HttpBasicClientImpl {

	timeout := tlsclient.DefaultClientTimeout
	urlParts, err := url.Parse(baseURL)
	if err != nil {
		slog.Error("Invalid URL")
		return nil
	}
	hostPort := urlParts.Host

	tlsClient := tls_client.NewTLSClient(hostPort, caCert, timeout)
	cl := NewHttpBasicTLSClientImpl(tlsClient, getForm)

	return cl
}

// NewHttpBasicTlsClient creates a new instance of the WoT compatible http-basic
// protocol binding client using the given TLS client.
//
//	tlsClient used for the server connection
//	getForm is the handler for return a form for invoking an operation. nil for default
func NewHttpBasicTLSClientImpl(
	tlsClient tlsclient.ITLSClient, getForm api.GetFormHandler) *HttpBasicClientImpl {

	thingID := httpbasic.HttpBasicClientModuleType + shortid.MustGenerate()
	cl := &HttpBasicClientImpl{
		HiveModuleBase: modules.NewHiveModuleBase(thingID, 0),
		getForm:        getForm,
		timeout:        tlsclient.DefaultClientTimeout,
		tlsClient:      tlsClient,
	}
	if cl.getForm == nil {
		cl.getForm = cl.GetDefaultForm
	}
	var _ api.IConnection = cl // interface check
	var _ api.IHiveModule = cl // interface check
	return cl
}

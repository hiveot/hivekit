package httpbasicapi

import (
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
	"github.com/hiveot/hivekit/go/modules/transports/httpserver"
	"github.com/hiveot/hivekit/go/modules/transports/httpserver/httpapi"
	"github.com/hiveot/hivekit/go/msg"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hivekit/go/wot"
	"github.com/hiveot/hivekit/go/wot/td"
	jsoniter "github.com/json-iterator/go"
	"github.com/teris-io/shortid"
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
	appConnectHandlerPtr atomic.Pointer[transports.ConnectionHandler]

	// authentication bearer token if authenticated
	bearerToken string

	//clientID string
	// Connection information such as clientID, cid, address, protocol etc
	cinfo transports.ConnectionInfo

	isConnected atomic.Bool

	// RPC timeout
	//timeout time.Duration
	// protected operations
	mux sync.RWMutex

	// getForm obtains the form for sending a request or notification
	// if nil, then the hiveot protocol envelope and URL are used as fallback
	getForm transports.GetFormHandler

	// destination for notifications, requests and responses.
	// This is intended to be the application module the client connects to.
	sink modules.IHiveModule

	// http2 client for posting messages
	tlsClient *httpapi.TLSClient
}

// ConnectWithClientCert creates a connection with the server using a client certificate for mutual authentication.
// The provided certificate must be signed by the server's CA.
//
//	kp is the key-pair used to the certificate validation
//	clientCert client tls certificate containing x509 cert and private key
//
// Returns nil if successful, or an error if connection failed
//
//	func (cl *HiveotSseClient) ConnectWithClientCert(kp keys.IHiveKey, clientCert *tls.Certificate) (err error) {
//		cl.mux.RLock()
//		defer cl.mux.RUnlock()
//		_ = kp
//		cl.tlsClient = tlsclient.NewTLSClient(cl.hostPort, clientCert, cl.caCert, cl.timeout)
//		return err
//	}

// ConnectWithToken sets the bearer token to use with requests.
func (cl *HttpBasicClient) ConnectWithToken(token string) error {

	// ensure disconnected (note that this resets retryOnDisconnect)
	cl.Disconnect()

	err := cl.SetBearerToken(token)
	if err != nil {
		return err
	}

	return err
}

// Disconnect from the server
func (cl *HttpBasicClient) Disconnect() {
	slog.Debug("HiveotSseClient.Disconnect",
		slog.String("clientID", cl.cinfo.ClientID),
	)

	cl.mux.Lock()
	defer cl.mux.Unlock()
	if cl.isConnected.Load() {
		cl.tlsClient.Close()
	}
}

// GetAppConnectHandler returns the application handler for connection status updates
func (cl *HttpBasicClient) GetAppConnectHandler() transports.ConnectionHandler {
	hPtr := cl.appConnectHandlerPtr.Load()
	return *hPtr
}

func (cl *HttpBasicClient) GetClientID() string {
	return cl.cinfo.ClientID
}

// GetConnectionInfo returns the client's connection details
func (cl *HttpBasicClient) GetConnectionInfo() transports.ConnectionInfo {
	return cl.cinfo
}

// GetDefaultForm return the default http form for the operation
// This simply returns nil for anything else than login, logout, ping or refresh.
func (cl *HttpBasicClient) GetDefaultForm(op, thingID, name string) (f *td.Form) {
	// login has its own URL as it is unauthenticated
	if op == wot.HTOpPing {
		href := httpbasic.HttpGetPingPath
		nf := td.NewForm(op, href)
		nf.SetMethodName(http.MethodGet)
		f = &nf
		//} else if op == wot.HTOpLogin {
		//	href := httpserver.HttpPostLoginPath
		//	nf := td.NewForm(op, href)
		//	nf.SetMethodName(http.MethodPost)
		//	f = &nf
		//} else if op == wot.HTOpLogout {
		//	href := httpserver.HttpPostLogoutPath
		//	nf := td.NewForm(op, href)
		//	nf.SetMethodName(http.MethodPost)
		//	f = &nf
		//} else if op == wot.HTOpRefresh {
		//	href := httpserver.HttpPostRefreshPath
		//	nf := td.NewForm(op, href)
		//	nf.SetMethodName(http.MethodPost)
		//	f = &nf
	}
	// everything else has no default form, so falls back to hiveot protocol endpoints
	return f
}

func (cl *HttpBasicClient) GetTlsClient() *http.Client {
	cl.mux.RLock()
	defer cl.mux.RUnlock()
	return cl.tlsClient.GetHttpClient()
}

// IsConnected return whether the return channel is connection, eg can receive data
func (cl *HttpBasicClient) IsConnected() bool {
	return cl.isConnected.Load()
}

// LoginWithForm invokes login using a form - temporary helper
// intended for testing a connection to a web server.
//
// This sets the bearer token for further requests. It requires the server
// to set a session cookie in response to the login.
//func (cl *HiveotSseClient) LoginWithForm(
//	password string) (newToken string, err error) {
//
//	// FIXME: does this client need a cookie jar???
//	formMock := url.Values{}
//	formMock.Add("loginID", cl.GetClientID())
//	formMock.Add("password", password)
//
//	var loginHRef string
//	f := cl.getForm(wot.HTOpLoginWithForm, "", "")
//	if f != nil {
//		loginHRef, _ = f.GetHRef()
//	}
//	loginURL, err := url.Parse(loginHRef)
//	if err != nil {
//		return "", err
//	}
//	if loginURL.Host == "" {
//		loginHRef = cl.fullURL + loginHRef
//	}
//
//	//PostForm should return a cookie that should be used in the http connection
//	if loginHRef == "" {
//		return "", errors.New("Login path not found in getForm")
//	}
//	resp, err := cl.httpClient.PostForm(loginHRef, formMock)
//	if err != nil {
//		return "", err
//	}
//
//	// get the session token from the cookie
//	//cookie := resp.Request.Header.Get("cookie")
//	cookie := resp.Header.Get("cookie")
//	kvList := strings.Split(cookie, ",")
//
//	for _, kv := range kvList {
//		kvParts := strings.SplitN(kv, "=", 2)
//		if kvParts[0] == "session" {
//			cl.bearerToken = kvParts[1]
//			break
//		}
//	}
//	if cl.bearerToken == "" {
//		slog.Error("No session cookie was received on login")
//	}
//	return cl.bearerToken, err
//}

// LoginWithPassword posts a login request to the TLS server using a login ID and
// password and obtain an auth token for use with SetBearerToken.
//
// FIXME: use a WoT standardized auth method
//
// If the connection fails then any existing connection is cancelled.
func (cl *HttpBasicClient) LoginWithPassword(password string) (newToken string, err error) {

	var method string
	var loginPath string

	clientID := cl.GetClientID()
	slog.Info("ConnectWithPassword",
		"clientID", clientID, "connectionID", cl.cinfo.ConnectionID)

	args := transports.UserLoginArgs{
		Login:    cl.GetClientID(),
		Password: password,
	}
	// is there a form for this? if not, fall back to HiveOT defaults
	f := cl.getForm(wot.HTOpLogin, "", "")
	if f == nil {
		slog.Warn("missing form for login operation. Using Http-basic defaults.")
	} else {
		method, _ = f.GetMethodName()
		loginPath = f.GetHRef() //
	}
	if method == "" {
		method = http.MethodPost
	}
	if loginPath == "" {
		loginPath = httpbasic.HttpPostLoginPath
	}
	dataJSON, _ := jsoniter.Marshal(args)
	outputRaw, _, _, err := cl.tlsClient.Send(
		method, loginPath, dataJSON, "", nil)

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

// Send a HTTPS method and return the http response.
//
// If token authentication is enabled then add the bearer token to the header
//
//	method: GET, PUT, POST, ...
//	reqPath: path to invoke
//	contentType of the payload or "" for default (application/json)
//	thingID optional path URI variable
//	name optional path URI variable containing affordance name
//	body contains the serialized payload
//	correlationID: optional correlationID header value
//
// This returns the raw serialized response data, a response message ID, return status code or an error
// func (cl *HttpBasicClient) Send(
// 	method string, methodPath string, body []byte) (
// 	resp []byte, headers http.Header, code int, err error) {

// 	if cl.httpClient == nil {
// 		err = fmt.Errorf("Send: '%s'. Client is not started", methodPath)
// 		return nil, nil, 0, err
// 	}
// 	// Caution! a double // in the path causes a 301 and changes post to get
// 	bodyReader := bytes.NewReader(body)
// 	serverURL := cl.cinfo.ConnectURL
// 	parts, _ := url.Parse(serverURL)
// 	parts.Scheme = "https"
// 	parts.Path = methodPath
// 	fullURL := parts.String()

// 	//fullURL := parts.cc.GetServerURL() + reqPath
// 	req, err := http.NewRequest(method, fullURL, bodyReader)
// 	if err != nil {
// 		err = fmt.Errorf("Send %s %s failed: %w", method, fullURL, err)
// 		return nil, nil, 0, err
// 	}

// 	// set the origin header to the intended destination without the path
// 	//parts, err := url.Parse(fullURL)
// 	origin := fmt.Sprintf("https://%s", parts.Host)
// 	req.Header.Set("Origin", origin)

// 	// set the authorization header
// 	if cl.bearerToken != "" {
// 		req.Header.Add("Authorization", "bearer "+cl.bearerToken)
// 	}

// 	// set other headers
// 	req.Header.Set("Content-Type", "application/json")
// 	req.Header.Set(httpserver.ConnectionIDHeader, cl.cinfo.ConnectionID)
// 	//if correlationID != "" {
// 	//	req.Header.Set(httpserver.CorrelationIDHeader, correlationID)
// 	//}
// 	for k, v := range cl.headers {
// 		req.Header.Set(k, v)
// 	}

// 	httpResp, err := cl.httpClient.Do(req)
// 	if err != nil {
// 		slog.Error(err.Error())
// 		return nil, nil, 0, err
// 	}

// 	respBody, err := io.ReadAll(httpResp.Body)
// 	// response body MUST be closed for clients
// 	_ = httpResp.Body.Close()
// 	httpStatus := httpResp.StatusCode

// 	if httpStatus == 401 {
// 		err = fmt.Errorf("%s", httpResp.Status)
// 	} else if httpStatus >= 400 && httpStatus < 500 {
// 		if respBody != nil {
// 			err = fmt.Errorf("%d (%s): %s", httpResp.StatusCode, httpResp.Status, respBody)
// 		} else {
// 			err = fmt.Errorf("%d (%s): Request failed", httpResp.StatusCode, httpResp.Status)
// 		}
// 	} else if httpStatus >= 500 {
// 		err = fmt.Errorf("Error %d (%s): %s", httpStatus, httpResp.Status, respBody)
// 		slog.Error("Send returned internal server error", "reqPath", methodPath, "err", err.Error())
// 	} else if err != nil {
// 		err = fmt.Errorf("Send: Error %s %s: %w", method, methodPath, err)
// 	}
// 	return respBody, httpResp.Header, httpStatus, err
// }

// pass the result of a http request to the registered response handler in the
// ResponseMessage envelope.
// func (cl *HttpBasicClient) handleRequestResult() {
//
// }

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
		httpserver.ThingIDURIVar:   thingID,
		httpserver.NameURIVar:      name,
		httpserver.OperationURIVar: req.Operation}
	reqPath := utils.Substitute(href, vars)
	contentType := "application/JSON"

	// send the request
	outputRaw, code, _, err := cl.tlsClient.Send(
		method, reqPath, []byte(inputJSON), contentType, nil)

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
			err = jsoniter.UnmarshalFromString(string(outputRaw), &resp.Value)
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
			err = jsoniter.UnmarshalFromString(string(outputRaw), &resp.Value)
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

// SendNotification is not supported in http-basic
func (cl *HttpBasicClient) SendNotification(msg *msg.NotificationMessage) error {
	return errors.New("HttpBasic doesn't support sending notifications")
}

// SetBearerToken sets the authentication bearer token to authenticate http requests.
func (cl *HttpBasicClient) SetBearerToken(token string) error {
	cl.mux.Lock()
	cl.bearerToken = token
	cl.mux.Unlock()
	return nil
}

// SetConnected sets the sub-protocol connection status
func (cl *HttpBasicClient) SetConnected(isConnected bool) {
	cl.isConnected.Store(isConnected)
}

// SetConnectHandler set the application handler for connection status updates
func (cl *HttpBasicClient) SetConnectHandler(cb transports.ConnectionHandler) {
	cl.appConnectHandlerPtr.Store(&cb)
}

// SetSink set the application module that handles async notifications, requests and responses
func (cl *HttpBasicClient) SetSink(sink modules.IHiveModule) {
	cl.mux.Lock()
	cl.sink = sink
	cl.mux.Unlock()
}

// NewHttpBasicClient creates a new instance of the http-basic protocol binding client.
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
	baseURL string, clientID string, caCert *x509.Certificate,
	sink modules.IHiveModule, getForm transports.GetFormHandler, timeout time.Duration) *HttpBasicClient {

	urlParts, err := url.Parse(baseURL)
	if err != nil {
		slog.Error("Invalid URL")
		return nil
	}
	hostPort := urlParts.Host

	tlsClient := httpapi.NewTLSClient(hostPort, nil, caCert, timeout)

	cl := HttpBasicClient{
		cinfo: transports.ConnectionInfo{
			CaCert:       caCert,
			ClientID:     clientID,
			ConnectionID: "http-" + shortid.MustGenerate(),
			ConnectURL:   baseURL,
			// ProtocolType: transports.ProtocolTypeHTTPBasic,
			Timeout: timeout,
		},
		getForm:   getForm,
		sink:      sink,
		tlsClient: tlsClient,
	}
	if cl.getForm == nil {
		cl.getForm = cl.GetDefaultForm
	}
	return &cl
}

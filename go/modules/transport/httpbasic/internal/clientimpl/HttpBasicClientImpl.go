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

	// The TD used to get the http URL for operations
	tdoc *td.TD

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

// SendNotification sends a notification to the server using a http request message
// Intended for devices that use RC to the server and receive requests oob.
func (cl *HttpBasicClientImpl) SendNotification(notif *msg.NotificationMessage) {

	// there is no form for this
	method := http.MethodPost
	payload, _ := jsoniter.Marshal(notif)
	contentType := "application/JSON"

	// this is a hiveot http-basic extension
	hrefPath := httpbasic.HttpBasicNotificationPath

	ctx := context.Background()
	_, code, _, err := cl.tlsClient.Send(
		ctx, method, hrefPath, nil, payload, contentType)

	if err != nil {
		slog.Warn("SendNotification failed", "code", code, "err", err.Error())
	}
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
	req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {

	var inputJSON string
	var method string
	var hrefPath string
	var form *td.Form
	var thingID = req.ThingID
	var name = req.Name

	if req.Operation == "" && req.CorrelationID == "" {
		err = fmt.Errorf("SendMessage: missing both operation and correlationID")
		slog.Error(err.Error())
		return err
	}

	// Inject URI variables for hrefs that use them:
	//  use + as wildcard for thingID to avoid a 404
	//  while not recommended, it is allowed to subscribe/observe all things
	if thingID == "" {
		thingID = "+"
	}
	//  use + as wildcard for affordance name to avoid a 404
	//  this should not happen very often but it is allowed
	// name := req.Name
	if name == "" {
		name = "+"
	}
	// substitute URI variables in the path, if any.
	// intended for use with http-basic forms.
	uriVars := map[string]string{
		td.UriVarThingID:   thingID,
		td.UriVarName:      name,
		td.UriVarOperation: req.Operation}

	// the getTD callback provides the method and URL to invoke for this operation.
	// use the hiveot fallback if not available
	// If the TD has no matching form then fall back to default well-known http basic href.
	if cl.tdoc != nil {
		form, _ = cl.tdoc.GetForm(req.Operation, req.Name, api.HttpBasicScheme, api.HttpBasicSubprotocol)
	}
	if form != nil {
		hrefURL, _ := form.ResolveHRef(cl.tdoc.Base, uriVars)
		hrefPath = hrefURL.Path
		method, _ = form.GetMethodName()
	} else {
		// fall back to the 'well known' hiveot request URL using uri variables
		// eg: /things/{op}/{id}/{name} or /hiveot/request
		method = http.MethodPost
		hrefPath = httpbasic.HttpBasicAffordanceOperationPath

		hrefPath = utils.Substitute(hrefPath, uriVars)
		inputJSON, _ = jsoniter.MarshalToString(req.Input)
	}

	contentType := "application/JSON"

	// send the request
	ctx, cancelFn := context.WithTimeout(context.Background(), cl.timeout)
	outputRaw, code, _, err := cl.tlsClient.Send(ctx,
		method, hrefPath, nil, []byte(inputJSON), contentType)
	cancelFn()

	// 1. error response
	if err != nil {
		return err
	}
	// follow the HTTP Basic specification
	if code == http.StatusOK || code == http.StatusNoContent {
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
		var resp *msg.ResponseMessage
		if len(outputRaw) == 0 {
			// request accepted but no response payload .. yet.
			// a response will be delivered out of band.
		} else {
			// a response payload is included. Looks like we're done.
			var tmp any
			err = jsoniter.Unmarshal(outputRaw, &tmp)
			resp = req.CreateResponse(tmp, err)
		}

		// pass a direct response to the application handler
		if resp != nil {
			_ = replyTo(resp)
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

// Set the clientID and authentication bearer token.
func (cl *HttpBasicClientImpl) SetAuthToken(clientID string, token string, secScheme string) error {

	_ = secScheme
	cl.bearerToken = token
	err := cl.tlsClient.SetAuthToken(clientID, token)
	return err
}

// Connect authenticating using a client certificate
func (cl *HttpBasicClientImpl) SetClientCert(clientCert *tls.Certificate) (err error) {
	return cl.tlsClient.SetClientCert(clientCert)
}

// Ignored as http clients dont receive notifications
func (cl *HttpBasicClientImpl) SetNotificationSink(sink api.IHiveModule, thingID ...string) {
	slog.Info("SetNotificationSink: HttpBasicClients doesn't receive notifications so not expecting any.",
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
// protocol binding client for use with the given TD.
//
// Users must use setAuthToken or SetClientCert to authenticate and Connect or Start
// to establish the connection.
//
// This uses TD forms to perform an operation.
//
//	tdoc is the TD to use for operations.
//	rootCAs to validate the server or nil to skip cert check
func NewHttpBasicClientImpl(tdoc *td.TD, rootCAs *x509.CertPool) *HttpBasicClientImpl {

	// FIXME: TD spec says base is optional and can vary per operation
	//
	urlParts, err := url.Parse(tdoc.Base)
	if err != nil {
		slog.Error("Invalid URL")
		return nil
	}
	hostPort := urlParts.Host

	tlsClient := tls_client.NewTLSClient(hostPort, rootCAs)
	if rootCAs == nil {
		tlsClient.SetSkipCertCheck(true)
	}
	cl := NewHttpBasicTLSClientImpl(tdoc, tlsClient)

	return cl
}

// NewHttpBasicTlsClient creates a new instance of the WoT compatible http-basic
// protocol binding client using the given configured TLS client.
//
//	tlsClient used for the server connection
//	getTD is the handler providing the TD for invoking an operation.
func NewHttpBasicTLSClientImpl(
	tdoc *td.TD, tlsClient tlsclient.ITLSClient) *HttpBasicClientImpl {

	thingID := httpbasic.HttpBasicClientModuleType + shortid.MustGenerate()
	cl := &HttpBasicClientImpl{
		HiveModuleBase: modules.NewHiveModuleBase(thingID, 0),
		tdoc:           tdoc,
		timeout:        tlsclient.DefaultClientTimeout,
		tlsClient:      tlsClient,
	}
	var _ api.IConnection = cl // interface check
	var _ api.IHiveModule = cl // interface check
	return cl
}

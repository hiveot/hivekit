package clientimpl

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
	"net/url"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/transport"
	"github.com/hiveot/hivekit/go/modules/transport/wss"
	"github.com/hiveot/hivekit/go/modules/transport/wss/internal"

	"github.com/teris-io/shortid"
)

// WssTransportClientImpl manages the connection to a websocket server.
// This implements the IConnection and IHiveModule interfaces.
//
// Usage 1 - wssclient is the sink for consumer and producer
//
//	requests:      consumer -> wssclient = wssserver -> producer
//	notifications: consumer <- wssclient = wssserver <- producer
//
// Usage 2 - wssserver is the sink for a server side consumer (gateway -> thing)
//
//	requests:      consumer -> wssserver = wssclient -> producer
//	notifications: consumer <- wssserver = wssclient <- producer
//
// This supports multiple message formats using a 'messageConverter'. The hiveot
// converts is a straight passthrough of RequestMessage and ResponseMessage, while
// the wotwssConverter maps the messages to the WoT websocket specification.
type WssTransportClientImpl struct {
	*modules.HiveModuleBase

	// authentication token
	bearerToken string

	caCert *x509.Certificate

	clientCert *tls.Certificate

	// connection ID set during connect
	cid string

	// The client connecting as
	clientID string

	// handler for sending connection notifications
	connectStatus api.ConnectionStatus
	// callback when connection changes
	connectHandler func(newStatus api.ConnectionStatus, c api.ITransportClient)

	// convert the request/response to the wss messaging protocol used
	encoder transport.IMessageEncoder

	// mutex for controlling writing and closing
	mux sync.RWMutex

	// the request & response channel handler
	// all responses are passed here to support response callbacks
	rnrChan *msg.RnRChan

	// underlying websocket connection
	wssConn     *websocket.Conn
	wssCancelFn context.CancelFunc

	wssURL string
	// wssPath string
}

// _onWssMessage processes the websocket message received from the server.
// This decodes the message into a request or response message and passes
// it to the application handler.
func (cl *WssTransportClientImpl) _onWssMessage(raw []byte) {
	var notif *msg.NotificationMessage
	var req *msg.RequestMessage
	var resp *msg.ResponseMessage
	clientID := cl.clientID

	// try to decode as notification first, then response, then request as websockets
	// do not carry metadata per request.
	notif, err := cl.encoder.DecodeNotification(raw)
	if err != nil {
		resp, err = cl.encoder.DecodeResponse(raw)
		if err != nil {
			req, err = cl.encoder.DecodeRequest(raw)
		}
	}
	if notif != nil {
		// client receives a notification message from the server
		// pass it on to the registered hook and sink
		go func() {
			cl.HiveModuleBase.HandleNotification(notif)
		}()
	} else if req != nil {
		var err error
		// client receives a request (using reverse connection)
		// pass it on to the linked producer.
		err = cl.ForwardRequest(req, func(resp *msg.ResponseMessage) error {
			// return the response to the caller
			err2 := cl.SendResponse(resp)
			return err2
		})
		// an error means the request could not be delivered
		if err != nil {
			resp := req.CreateErrorResponse(err)
			_ = cl.SendResponse(resp)
		}
	} else if resp != nil {
		// client receives a response message
		// pass it on to the waiting consumer
		handled := cl.rnrChan.HandleResponse(resp, cl.GetTimeout())
		if !handled {
			slog.Error("_onWssMessage: received response but no matching request",
				"correlationID", resp.CorrelationID,
				"op", resp.Operation,
				"name", resp.Name,
				"clientID", clientID,
			)
		}
	} else {
		slog.Warn("_onWssMessage: Message is not a valid request, response or notification, request or response",
			"raw", string(raw))
		return
	}
}

// _send publishes a message over websockets
func (cl *WssTransportClientImpl) _send(wssMsg []byte) (err error) {
	// websockets do not allow concurrent writes
	cl.mux.Lock()
	defer cl.mux.Unlock()

	if cl.wssConn == nil {
		err := fmt.Errorf("_send: Can't send. Not connected")
		return err
	}

	if cl.connectStatus == api.StatusConnecting {
		// TODO: should we wait for a bit while connecting?
		return fmt.Errorf("_send: Not connected. Connecting.")
	} else if cl.connectStatus != api.StatusConnected {
		return fmt.Errorf("_send: Not connected")
	}

	// Use WriteMessage because the message is already JSON serialized
	err = cl.wssConn.WriteMessage(websocket.TextMessage, wssMsg)
	if err != nil {
		err = fmt.Errorf("WssClient._send write error: %s", err)
	}
	return err
}

// websocket connection status handler - this uses mux lock for critical section
func (cl *WssTransportClientImpl) _setConnectionStatus(newStatus api.ConnectionStatus, err error) {

	cl.mux.Lock()
	oldStatus := cl.connectStatus
	cl.connectStatus = newStatus
	ch := cl.connectHandler
	cl.mux.Unlock()

	if newStatus == oldStatus {
		return
	} else if oldStatus == api.StatusClosed && newStatus == api.StatusLost {
		// already closed, don't send status lost
		return
	} else if newStatus == api.StatusLost {
		slog.Info("_setConnectionStatus WSS client connection lost", "status", newStatus)
		// fail all outstanding RnR requests
		cl.rnrChan.CloseAll()
	}

	// notify upstream of connect, disconnect or lost
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

// AuthenticateWithClientCert sets the authentication credentials to the client certificate.
func (cl *WssTransportClientImpl) AuthenticateWithClientCert(clientCert *tls.Certificate) (err error) {
	status := cl.GetConnectionStatus()
	if status == api.StatusConnected || status == api.StatusConnecting {
		return fmt.Errorf("AuthenticateWithClientCert: Connection in progress.")
	}
	// tell the client to use the certificate
	cl.clientCert = clientCert

	//--- verify the client certificate against the CA and extract the clientID
	// if a client cert is given then test if it is valid for our CA.
	// this detects problems with certs that can be hard to track down
	x509Cert, err := x509.ParseCertificate(clientCert.Certificate[0])
	if err == nil {
		caCertPool := x509.NewCertPool()
		caCertPool.AddCert(cl.caCert)

		// cert subject is clientID
		cl.clientID = x509Cert.Subject.CommonName

		// verify the validity of this certificate against the CA
		// without this one can spend a long time figuring out why the connection fails.
		opts := x509.VerifyOptions{
			Roots:     caCertPool,
			KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		}
		_, err = x509Cert.Verify(opts)
	}

	// err = cl.tlsClient.AuthenticateWithClientCert(clientCert)
	// if err != nil {
	// return err
	// }
	// err = cl.Connect()
	return err
}

// AuthenticateWithForm authenticates the client using the method described in the form.
//
// This currently only supports bearer token authentication.
//
// This determine which auth schema the TD describes, obtains the credentials
// and injects the authentication credentials according to the TDI schema.
// This returns an error if the schema isn't supported or is not compatible.
func (cl *WssTransportClientImpl) AuthenticateWithForm(tdDoc *td.TD,
	getCredentials api.GetCredentials) error {

	var secScheme td.SecurityScheme

	// for now just assume its bearer token, just to get it working
	clientID, secret, schemeName, err := getCredentials(tdDoc.ID)
	if err != nil {
		slog.Warn("AuthenticateWithForm: No credentials for thing. Continuing", "thingID", tdDoc.ID)
	}
	secScheme, err = tdDoc.GetSecurityScheme()
	if err != nil {
		return err
	}
	if secScheme.Scheme == td.SecSchemeNoSec {
		err = cl.AuthenticateWithToken(clientID, "")
	} else if schemeName != secScheme.Scheme && schemeName != "" && schemeName != td.SecSchemeAuto {
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

// AuthenticateWithToken sets the token credentials to use in Connect
func (cl *WssTransportClientImpl) AuthenticateWithToken(clientID string, token string) error {

	status := cl.GetConnectionStatus()
	if status == api.StatusConnected || status == api.StatusConnecting {
		return fmt.Errorf("AuthenticateWithToken: Connection in progress.")
	}
	cl.clientID = clientID
	cl.bearerToken = token
	return nil
}

// Disconnect from the server
func (cl *WssTransportClientImpl) Close() {

	// set status to closed first to avoid a reconnect
	cl._setConnectionStatus(api.StatusClosed, nil)

	cl.mux.Lock()
	defer cl.mux.Unlock()
	if cl.wssCancelFn != nil {
		cl.wssCancelFn()
		cl.wssCancelFn = nil
	}
}

// Establish a websocket connection using the previously setup credentials
// If a connection attempt is in progress then wait.
func (cl *WssTransportClientImpl) Connect() error {
	status := cl.GetConnectionStatus()

	if status == api.StatusConnected {
		return nil
	} else if status == api.StatusConnecting {
		return fmt.Errorf("Busy connecting")
	}

	// differentiate connections from the same client
	if cl.cid == "" {
		cl.cid = cl.GetThingID()
	}

	urlParts, err := url.Parse(cl.wssURL)
	if err != nil {
		return err
	}
	hostPort := urlParts.Host
	wssCancelFn, wssConn, err := ConnectWSS(
		cl.clientID, hostPort, urlParts.Path, cl.cid,
		cl.bearerToken, cl.clientCert, cl.caCert,
		cl._setConnectionStatus,
		cl._onWssMessage)

	cl.mux.Lock()
	cl.wssCancelFn = wssCancelFn
	cl.wssConn = wssConn
	cl.mux.Unlock()

	return err
}

// GetClientID returns the client's connection details
func (cl *WssTransportClientImpl) GetClientID() string {
	return cl.clientID
}

// // GetConnectionID returns the client's connection details
func (cl *WssTransportClientImpl) GetConnectionID() string {
	return cl.cid
}

// // GetConnectionStatus returns the current connection status
func (cl *WssTransportClientImpl) GetConnectionStatus() api.ConnectionStatus {
	cl.mux.RLock()
	defer cl.mux.RUnlock()
	stat := cl.connectStatus
	return stat
}

// HandleNotification receives an incoming notification from a producer
// and sends it to the server.
func (m *WssTransportClientImpl) HandleNotification(notif *msg.NotificationMessage) {
	// Can't use HiveModuleBase.HandleNotification as it forwards the notification
	// to the registered notification sink. Instead it should go to the server.
	m.SendNotification(notif)
}

// Clients receives a request
// - reconnect actions are handled here
// - other requests (like subscribe) are send to the server
func (cl *WssTransportClientImpl) HandleRequest(request *msg.RequestMessage, replyTo msg.ResponseHandler) error {
	if request.ThingID == cl.GetThingID() {
		if request.Operation == td.OpInvokeAction && request.Name == api.ClientConnectAction {
			err := cl.Connect()
			resp := request.CreateResponse(cl.GetConnectionStatus(), err)
			return replyTo(resp)
		} else {
			return fmt.Errorf("HandleRequest: invalid request op='%s', name='%s'",
				request.Operation, request.Name)
		}
	}
	err := cl.SendRequest(request, replyTo)
	return err
}

// SendNotification Device posts a notification over to the server
// This serializes the notification and sends it to the server.
func (cl *WssTransportClientImpl) SendNotification(notif *msg.NotificationMessage) {
	slog.Info("SendNotification",
		slog.String("clientID", cl.clientID),
		slog.String("correlationID", notif.CorrelationID),
		slog.String("affordance", string(notif.AffordanceType)),
		slog.String("thingID", notif.ThingID),
		slog.String("name", notif.Name),
	)
	// convert the operation into a protocol message
	wssMsg, err := cl.encoder.EncodeNotification(notif)
	if err != nil {
		slog.Error("SendNotification: unknown affordance", "affordanceType", notif.AffordanceType)
	}
	err = cl._send(wssMsg)
	if err != nil {
		slog.Warn("SendNotification failed", "clientID", cl.clientID, "err", err.Error())
	}
}

// SendRequest send a request message over websockets
// This transforms the request to the protocol message and sends it to the server.
func (cl *WssTransportClientImpl) SendRequest(
	req *msg.RequestMessage, replyTo msg.ResponseHandler) error {
	slog.Info("SendRequest",
		slog.String("clientID", cl.clientID),
		slog.String("correlationID", req.CorrelationID),
		slog.String("operation", req.Operation),
		slog.String("thingID", req.ThingID),
		slog.String("name", req.Name),
	)

	if req.CorrelationID == "" {
		req.CorrelationID = shortid.MustGenerate()
	}
	// convert the operation into a protocol message
	wssMsg, err := cl.encoder.EncodeRequest(req)
	if err != nil {
		slog.Error("SendRequest: unknown request", "op", req.Operation)
		return err
	}
	if replyTo == nil {
		// responses are received asynchronously
		err = cl._send(wssMsg)
		return err
	}

	// a response handler is provided, callback when the response is received
	cl.rnrChan.Open(req.CorrelationID)
	err = cl._send(wssMsg)

	if err != nil {
		cl.rnrChan.Close(req.CorrelationID)
		slog.Warn("SendRequest ->: error in sending request",
			"dThingID", req.ThingID,
			"name", req.Name,
			"correlationID", req.CorrelationID,
			"err", err.Error())
		return err
	}
	// FIXME: should this run async in the background?
	hasResponse, resp := cl.rnrChan.WaitForResponse(req.CorrelationID, cl.GetTimeout())
	if hasResponse {
		err = replyTo(resp)
	} else {
		err = fmt.Errorf("No response received")
	}
	return err
}

// SendResponse send a response message over websockets
// This transforms the response to the protocol message and sends it to the server.
// Responses without correlationID are subscription notifications.
func (cl *WssTransportClientImpl) SendResponse(resp *msg.ResponseMessage) error {
	clientID := cl.clientID
	errMsg := ""
	if resp.Error != nil {
		errMsg = resp.Error.String()
	}
	slog.Debug("SendResponse",
		slog.String("operation", resp.Operation),
		slog.String("clientID", clientID),
		slog.String("thingID", resp.ThingID),
		slog.String("name", resp.Name),
		slog.String("error", errMsg),
		slog.String("correlationID", resp.CorrelationID),
	)

	// convert the operation into a protocol message
	wssMsg, err := cl.encoder.EncodeResponse(resp)
	err = cl._send(wssMsg)
	return err
}

// SetConnectHandler sets the callback to invoke when the connection status changes
func (cl *WssTransportClientImpl) SetConnectHandler(
	h func(newStatus api.ConnectionStatus, c api.ITransportClient)) {
	cl.mux.Lock()
	defer cl.mux.Unlock()
	cl.connectHandler = h
}

// Start the module but do not yet connect.
//
// Intended for use by the factory as the factory provides a clientID/token or client
// certificate.
//
// Most users will use AuthenticateWithToken() followed by Connect() instead.
func (cl *WssTransportClientImpl) Start() error {
	err := cl.Connect()
	return err
}

// Module stop
func (cl *WssTransportClientImpl) Stop() {
	cl.Close()
}

// NewHiveotWssTransportClient creates a new instance of the hiveot websocket client.
//
// This uses the Hiveot passthrough message converter.
// Users must use AuthenticateWithToken to authenticate and connect.
//
//	wssURL is the full websocket connection URL including path
//	caCert is the server CA for TLS connection validation
func NewHiveotWssClientImpl(
	wssURL string, caCert *x509.Certificate) *WssTransportClientImpl {

	timeout := msg.DefaultRnRTimeout
	thingID := wss.HiveotWebsocketClientModuleType + shortid.MustGenerate()

	cl := WssTransportClientImpl{
		HiveModuleBase: modules.NewHiveModuleBase(thingID, timeout),
		caCert:         caCert,
		// hiveot uses its own standardized RRN messages
		encoder: transport.NewRRNJsonEncoder(),
		rnrChan: msg.NewRnRChan(),
		wssURL:  wssURL,
	}
	return &cl
}

// NewWotWssTransportClient creates a new instance of the WoT compatible websocket client.
//
// Users must use AuthenticateWithToken to authenticate and connect.
//
//	wssURL is the full websocket connection URL
//	caCert is the server CA for TLS connection validation
//	timeout is the maximum connection wait time. 0 for default.
//	ch is the connection callback handler, nil to ignore
func NewWotWssClientImpl(
	wssURL string, caCert *x509.Certificate) *WssTransportClientImpl {

	timeout := msg.DefaultRnRTimeout
	thingID := wss.HiveotWebsocketClientModuleType + shortid.MustGenerate()

	cl := &WssTransportClientImpl{
		HiveModuleBase: modules.NewHiveModuleBase(thingID, timeout),
		caCert:         caCert,
		encoder:        internal.NewWotWssMsgEncoder(),
		rnrChan:        msg.NewRnRChan(),
		wssURL:         wssURL,
	}
	var _ api.ITransportClient = cl // interface check
	return cl
}

package clientimpl

import (
	"context"
	"crypto/x509"
	"fmt"
	"log/slog"
	"net/url"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/cells/transport"
	"github.com/hiveot/hivekit/go/cells/transport/wss"
	"github.com/hiveot/hivekit/go/cells/transport/wss/internal"

	"github.com/teris-io/shortid"
)

// WssTransportClientImpl manages the connection to a websocket server.
// This implements the IConnection and IHiveCell interfaces.
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
	*transport.TransportClientBase

	rootCAs *x509.CertPool

	// convert the request/response to the wss messaging protocol used
	encoder transport.IMessageEncoder

	// mutex for controlling writing and closing
	writeMux sync.RWMutex

	// the request & response channel handler
	// all responses are passed here to support response callbacks
	rnrChan *msg.RnRChan

	// underlying websocket connection
	wssConn     *websocket.Conn
	wssCancelFn context.CancelFunc

	wssURL string
	// wssPath string
}

// _onWssClientMessage processes the websocket message received from the server.
// This decodes the message into a request or response message and passes
// it to the application handler.
//
// TODO: _onClientMessage might go into a ClientConnectionBase and reused by grpc, wss, ...
//
//	as they only differ by encoder.
func (cl *WssTransportClientImpl) _onWssClientMessage(raw []byte) {
	var err error
	msgType := cl.encoder.DetermineMessageType(raw)
	if msgType == "" {
		slog.Warn("_onWssClientMessage: Wss message is has no messageType", "raw", string(raw))
		return
	}
	switch msgType {
	case msg.MessageTypeNotification:
		var notif *msg.NotificationMessage
		notif, err = cl.encoder.DecodeNotification("", raw)
		if err == nil {
			// client receives a notification message from the server
			// pass it on to the registered hook and sink
			go func() {
				cl.HiveCellBase.HandleNotification(notif)
			}()
			return
		}

	case msg.MessageTypeRequest:
		var req *msg.RequestMessage
		req, err = cl.encoder.DecodeRequest("", raw)
		if err == nil {
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
			return
		}

	case msg.MessageTypeResponse:
		var resp *msg.ResponseMessage
		resp, err = cl.encoder.DecodeResponse("", raw)
		if err == nil {
			// client receives a response message
			// pass it on to the waiting consumer
			handled := cl.rnrChan.HandleResponse(resp, cl.GetTimeout())
			if !handled {
				slog.Error("_onWssClientMessage: received response but no matching request",
					"correlationID", resp.CorrelationID,
					"op", resp.Operation,
					"name", resp.Name,
					"clientID", cl.GetClientID(),
				)
			}
		}
	default:
		err = fmt.Errorf("Unknown message type '%s'", msgType)
	}
	if err != nil {
		slog.Warn("_onWssClientMessage: Failed to decode message", "msgType", msgType, "err", err.Error())
	}
}

// _send publishes a message over websockets
func (cl *WssTransportClientImpl) _send(wssMsg []byte) (err error) {
	cstatus := cl.GetConnectionStatus()

	// websockets do not allow concurrent writes
	cl.writeMux.Lock()
	defer cl.writeMux.Unlock()

	if cl.wssConn == nil {
		err := fmt.Errorf("_send: No websocket connection. Status: %s", cstatus)
		return err
	}

	if cstatus == api.StatusConnecting {
		// TODO: should we wait for a bit while connecting?
		return fmt.Errorf("_send: Not connected. Connecting.")
	} else if cstatus != api.StatusConnected {
		return fmt.Errorf("_send: Not connected")
	}

	// Use WriteMessage because the message is already JSON serialized
	err = cl.wssConn.WriteMessage(websocket.TextMessage, wssMsg)
	if err != nil {
		err = fmt.Errorf("WssClient._send write error: %s", err)
	}
	return err
}

// update the connection status and publish an notification if it differs from the last status
// a 'lost' status is ignored if the current status is set to closed as it was intentional.
func (cl *WssTransportClientImpl) _setConnectionStatus(newStatus api.ConnectionStatus, err error) {

	if newStatus == api.StatusLost {
		slog.Info("_setConnectionStatus SseCl client connection lost", "status", newStatus)
		// fail all outstanding RnR requests
		cl.rnrChan.CloseAll()
	}
	cl.TransportClientBase.SetConnectionStatus(newStatus, err)
}

// Disconnect from the server
func (cl *WssTransportClientImpl) Close() {

	// set status to closed first to avoid a reconnect
	cl._setConnectionStatus(api.StatusClosed, nil)

	// wait until any writes are complete
	cl.writeMux.Lock()
	defer cl.writeMux.Unlock()
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
	// if cl.cid == "" {
	// 	cl.cid = cl.GetThingID()
	// }

	urlParts, err := url.Parse(cl.wssURL)
	if err != nil {
		return err
	}
	cid := cl.GetConnectionID()
	clientID := cl.GetClientID()
	clientCert := cl.GetClientCert()
	bearerToken, secScheme := cl.GetAuthToken()
	_ = secScheme
	hostPort := urlParts.Host
	wssCancelFn, wssConn, err := ConnectWSS(
		clientID, hostPort, urlParts.Path, cid,
		bearerToken, clientCert, cl.rootCAs,
		cl._setConnectionStatus,
		cl._onWssClientMessage)

	cl.writeMux.Lock()
	cl.wssCancelFn = wssCancelFn
	cl.wssConn = wssConn
	cl.writeMux.Unlock()

	return err
}

// HandleNotification receives an incoming notification from a producer
// and sends it to the server.
func (m *WssTransportClientImpl) HandleNotification(notif *msg.NotificationMessage) {
	// Can't use HiveCellBase.HandleNotification as it forwards the notification
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
		slog.String("clientID", cl.GetClientID()),
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
		slog.Warn("SendNotification failed",
			"clientID", cl.GetClientID(),
			"err", err.Error())
	}
}

// SendRequest send a request message over websockets
// This transforms the request to the protocol message and sends it to the server.
func (cl *WssTransportClientImpl) SendRequest(
	req *msg.RequestMessage, replyTo msg.ResponseHandler) error {
	slog.Info("SendRequest",
		slog.String("clientID", cl.GetClientID()),
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
	clientID := cl.GetClientID()
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

// Start the client but do not yet connect.
//
// Intended for use by the factory as the factory provides a clientID/token or client
// certificate.
//
// Most users will use Connect()
func (cl *WssTransportClientImpl) Start() error {
	err := cl.Connect()
	return err
}

// Stop the client and disconnect if connected
func (cl *WssTransportClientImpl) Stop() {
	cl.Close()
}

// NewHiveotWssTransportClient creates a new instance of the hiveot websocket client.
//
// This uses the highly efficient Hiveot passthrough message converter.
// Users must use setAuthToken or SetClientCert to authenticate and Connect or Start
// to establish the connection.
//
//	wssURL is the full websocket connection URL including path
//	rootCAs are the CA's for TLS connection validation
func NewHiveotWssClientImpl(
	wssURL string, rootCAs *x509.CertPool) *WssTransportClientImpl {

	timeout := msg.DefaultRnRTimeout
	thingID := wss.HiveotWebsocketClientCellType + shortid.MustGenerate()

	cl := WssTransportClientImpl{
		TransportClientBase: transport.NewTransportClientBase(thingID, rootCAs, timeout),
		rootCAs:             rootCAs,
		// hiveot uses its own standardized RRN messages
		encoder: transport.NewRRNJsonEncoder(),
		rnrChan: msg.NewRnRChan(),
		wssURL:  wssURL,
	}
	return &cl
}

// NewWotWssTransportClient creates a new instance of the WoT compatible websocket client.
//
// Users must use setAuthToken or SetClientCert to authenticate and Connect or Start
// to establish the connection.
//
//	wssURL is the full websocket connection URL
//	caCerootCAs are the CA's for TLS connection validation
//	timeout is the maximum connection wait time. 0 for default.
//	ch is the connection callback handler, nil to ignore
func NewWotWssClientImpl(
	wssURL string, rootCAs *x509.CertPool) *WssTransportClientImpl {

	timeout := msg.DefaultRnRTimeout
	thingID := wss.HiveotWebsocketClientCellType + shortid.MustGenerate()

	cl := &WssTransportClientImpl{
		TransportClientBase: transport.NewTransportClientBase(thingID, rootCAs, timeout),
		rootCAs:             rootCAs,
		encoder:             internal.NewWotWssMsgEncoder(),
		rnrChan:             msg.NewRnRChan(),
		wssURL:              wssURL,
	}
	var _ api.ITransportClient = cl // interface check
	return cl
}

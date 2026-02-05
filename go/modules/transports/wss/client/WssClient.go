package wssclient

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/modules/transports/direct"
	tlsclient "github.com/hiveot/hivekit/go/modules/transports/httpserver/client"
	"github.com/hiveot/hivekit/go/modules/transports/wss/converter"
	"github.com/hiveot/hivekit/go/msg"

	"github.com/teris-io/shortid"
)

// WssClient manages the connection to a websocket server.
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
type WssClient struct {
	modules.HiveModuleBase

	// authentication token
	bearerToken string

	caCert *x509.Certificate

	// handler for requests send by clients
	connectHandler transports.ConnectionHandler

	isConnected atomic.Bool
	// lastError   atomic.Pointer[error]

	maxReconnectAttempts int // 0 for indefinite

	// convert the request/response to the wss messaging protocol used
	msgConverter transports.IMessageConverter

	// mutex for controlling writing and closing
	mux sync.RWMutex

	retryOnDisconnect atomic.Bool

	// the request & response channel handler
	// all responses are passed here to support response callbacks
	rnrChan *msg.RnRChan

	// send/receive timeout to use
	timeout time.Duration

	// Destination for incoming requests?
	// FIXME: do clients have sinks?
	// server -> client -> ?
	// app module [request] -> client -> server -> [request] -> module
	//              [notif] <- client <- server <- [notif] <- module

	// http2 client for posting messages
	tlsClient transports.ITlsClient

	// underlying websocket connection
	wssConn     *websocket.Conn
	wssCancelFn context.CancelFunc

	wssURL  string
	wssPath string
}

// websocket connection status handler
func (cl *WssClient) _onConnectionChanged(connected bool, err error) {

	cl.isConnected.Store(connected)
	if cl.connectHandler != nil {
		cl.connectHandler(connected, cl, err)
	}
	// if retrying is enabled then try on disconnect
	if !connected && cl.retryOnDisconnect.Load() {
		cl.Reconnect()
	}
}

// _send publishes a message over websockets
func (cl *WssClient) _send(wssMsg []byte) (err error) {
	if !cl.isConnected.Load() {
		// note, it might be trying to reconnect in the background
		err := fmt.Errorf("_send: Not connected to the hub")
		return err
	}
	// websockets do not allow concurrent writes
	cl.mux.Lock()
	defer cl.mux.Unlock()
	// Use WriteMessage because the message is already JSON serialized
	err = cl.wssConn.WriteMessage(websocket.TextMessage, wssMsg)
	if err != nil {
		err = fmt.Errorf("WssClient._send write error: %s", err)
	}
	return err
}

// Disconnect from the server
func (cl *WssClient) Close() {
	slog.Info("Close",
		slog.String("clientID", cl.tlsClient.GetClientID()),
	)
	// dont try to reconnect
	cl.retryOnDisconnect.Store(false)

	cl.mux.Lock()
	defer cl.mux.Unlock()
	if cl.wssCancelFn != nil {
		cl.wssCancelFn()
		cl.wssCancelFn = nil
	}
}

// ConnectWithToken attempts to establish a websocket connection using a valid auth token
// If a connection exists it is closed first.
func (cl *WssClient) ConnectWithToken(clientID string, token string, ch transports.ConnectionHandler) error {

	// ensure disconnected (note that this resets retryOnDisconnect)
	cl.Close()
	cl.connectHandler = ch
	cl.bearerToken = token
	// the clientID is the moduleID so set it now
	cl.SetModuleID(clientID)
	cl.tlsClient.ConnectWithToken(clientID, token)
	hostPort := cl.tlsClient.GetHostPort()
	wssCancelFn, wssConn, err := ConnectWSS(
		clientID, hostPort, cl.wssPath, cl.bearerToken, nil, cl.caCert,
		cl._onConnectionChanged, cl.HandleWssMessage)

	cl.mux.Lock()
	cl.wssCancelFn = wssCancelFn
	cl.wssConn = wssConn
	cl.mux.Unlock()

	// even if connection failed right now, enable retry
	cl.retryOnDisconnect.Store(true)

	return err
}

// // GetClientID returns the client's connection details
func (cl *WssClient) GetClientID() string {
	return cl.tlsClient.GetClientID()
}

// // GetConnectionID returns the client's connection details
func (cl *WssClient) GetConnectionID() string {
	return cl.tlsClient.GetConnectionID()
}

func (cl *WssClient) GetTlsClient() *http.Client {
	cl.mux.RLock()
	defer cl.mux.RUnlock()
	return cl.tlsClient.GetHttpClient()
}

// HandleNotification receives an incoming notification from a producer
// and sends it to the server.
func (m *WssClient) HandleNotification(notif *msg.NotificationMessage) {
	// Can't use HiveModuleBase.HandleNotification as it forwards the notification
	// to the registered notification sink.
	m.SendNotification(notif)
}

// clients send requests to the server
func (cl *WssClient) HandleRequest(request *msg.RequestMessage, replyTo msg.ResponseHandler) error {
	err := cl.SendRequest(request, replyTo)
	return err
}

// HandleWssMessage processes the websocket message received from the server.
// This decodes the message into a request or response message and passes
// it to the application handler.
func (cl *WssClient) HandleWssMessage(raw []byte) {
	var notif *msg.NotificationMessage
	var req *msg.RequestMessage
	var resp *msg.ResponseMessage
	clientID := cl.tlsClient.GetClientID()

	// // for testing:
	// var jsonObj any
	// err := jsoniter.Unmarshal(raw, &jsonObj)
	// if err != nil {
	// 	slog.Error("HandleWssMessage: failed to decode JSON",
	// 		"clientID", cc.cinfo.ClientID,
	// 		"err", err.Error(),
	// 		"raw", string(raw))
	// 	return
	// }

	// try to decode as notification first, then response, then request as

	// both non-agents and agents receive responses
	notif = cl.msgConverter.DecodeNotification(raw)
	if notif == nil {
		resp = cl.msgConverter.DecodeResponse(raw)
		if resp == nil {
			req = cl.msgConverter.DecodeRequest(raw)
		}
	}
	if notif != nil {
		// client receives a notification message from the server
		// pass it on to the registered hook and sink
		go cl.HiveModuleBase.HandleNotification(notif)
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
		handled := cl.rnrChan.HandleResponse(resp, cl.timeout)
		if !handled {
			slog.Error("HandleWssMessage: received response but no matching request",
				"correlationID", resp.CorrelationID,
				"op", resp.Operation,
				"name", resp.Name,
				"clientID", clientID,
			)
		}
	} else {
		slog.Warn("HandleWssMessage: Message is not a valid notification, request or response",
			"raw", string(raw))
		return
	}
}

// IsConnected return whether the return channel is connection, eg can receive data
func (cl *WssClient) IsConnected() bool {
	return cl.isConnected.Load()
}

// Reconnect attempts to re-establish a dropped connection using the last token
// This uses an increasing backoff period up to 15 seconds, starting random between 0-2 seconds
func (cl *WssClient) Reconnect() {
	var err error
	var backoffDuration time.Duration = time.Duration(rand.Uint64N(uint64(time.Second * 2)))

	clientID := cl.tlsClient.GetClientID()
	for i := 0; cl.maxReconnectAttempts == 0 || i < cl.maxReconnectAttempts; i++ {
		slog.Warn("Reconnecting attempt",
			slog.String("clientID", clientID),
			slog.Int("i", i))
		err = cl.ConnectWithToken(clientID, cl.bearerToken, cl.connectHandler)
		if err == nil {
			break
		}
		// retry until max repeat is reached, disconnect is called or authorization failed
		if !cl.retryOnDisconnect.Load() {
			break
		}
		if errors.Is(err, transports.UnauthorizedError) {
			break
		}
		// the connection timeout doesn't seem to work for some reason
		//
		time.Sleep(backoffDuration)
		// slowly wait longer until 10 sec. FIXME: use random
		if backoffDuration < time.Second*15 {
			backoffDuration += time.Second
		}
	}
	if err != nil {
		slog.Warn("Reconnect failed: ", "err", err.Error())
	}
}

// SendNotification Agent posts a notification over websockets
// This passes the notification as-is as a payload.
//
// This posts the JSON-encoded NotificationMessage on the well-known hiveot notification href.
// In WoT Agents are typically a server, not a client, so this is intended for
// agents that use connection-reversal.
func (cl *WssClient) SendNotification(notif *msg.NotificationMessage) {
	clientID := cl.tlsClient.GetClientID()
	slog.Info("SendNotification",
		slog.String("clientID", clientID),
		slog.String("correlationID", notif.CorrelationID),
		slog.String("operation", notif.Operation),
		slog.String("thingID", notif.ThingID),
		slog.String("name", notif.Name),
	)
	// convert the operation into a protocol message
	wssMsg, err := cl.msgConverter.EncodeNotification(notif)
	if err != nil {
		slog.Error("SendNotification: unknown operation", "op", notif.Operation)
	}
	err = cl._send(wssMsg)
	if err != nil {
		slog.Warn("SendNotification failed",
			"clientID", clientID,
			"err", err.Error())
	}
}

// SendRequest send a request message over websockets
// This transforms the request to the protocol message and sends it to the server.
func (cl *WssClient) SendRequest(
	req *msg.RequestMessage, replyTo msg.ResponseHandler) error {
	clientID := cl.tlsClient.GetClientID()
	slog.Debug("SendRequest",
		slog.String("clientID", clientID),
		slog.String("correlationID", req.CorrelationID),
		slog.String("operation", req.Operation),
		slog.String("thingID", req.ThingID),
		slog.String("name", req.Name),
	)

	if req.CorrelationID == "" {
		req.CorrelationID = shortid.MustGenerate()
	}
	// convert the operation into a protocol message
	wssMsg, err := cl.msgConverter.EncodeRequest(req)
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
	hasResponse, resp := cl.rnrChan.WaitForResponse(req.CorrelationID, cl.timeout)
	if hasResponse {
		err = replyTo(resp)
	}
	return err
}

// SendResponse send a response message over websockets
// This transforms the response to the protocol message and sends it to the server.
// Responses without correlationID are subscription notifications.
func (cl *WssClient) SendResponse(resp *msg.ResponseMessage) error {
	clientID := cl.tlsClient.GetClientID()
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
	wssMsg, err := cl.msgConverter.EncodeResponse(resp)
	err = cl._send(wssMsg)
	return err
}

// Change the default timeout for sending and waiting for messages
func (cl *WssClient) SetTimeout(timeout time.Duration) {
	cl.tlsClient.SetTimeout(timeout)
	cl.timeout = timeout
}

// NewHiveotWssClient creates a new instance of the hiveot websocket client.
//
// This uses the Hiveot passthrough message converter.
//
//	wssURL is the full websocket connection URL including path
//	clientID is the authentication ID of the consumer or agent
//	caCert is the server CA for TLS connection validation
//	timeout is the maximum connection wait time
func NewHiveotWssClient(
	wssURL string, caCert *x509.Certificate,
	timeout time.Duration) *WssClient {

	// ensure the URL has port as 443 is not valid for this
	urlParts, err := url.Parse(wssURL)
	if err != nil {
		slog.Error("Invalid URL")
		return nil
	}
	hostPort := urlParts.Host
	wssPath := urlParts.Path

	if timeout == 0 {
		timeout = transports.DefaultRpcTimeout
	}

	tlsClient := tlsclient.NewTLSClient(hostPort, nil, caCert, timeout)

	cl := WssClient{
		maxReconnectAttempts: 0,
		// hiveot uses its own standardized RRN messages
		msgConverter: direct.NewPassthroughMessageConverter(),
		rnrChan:      msg.NewRnRChan(),
		timeout:      timeout,
		tlsClient:    tlsClient,
		wssPath:      wssPath,
		wssURL:       wssURL,
	}
	//cl.Init(fullURL, clientID, clientCert, caCert, getForm, timeout)
	return &cl
}

// NewWotWssClient creates a new instance of the WoT compatible websocket client.
//
// messageConverter offers the ability to use any websocket message format that
// can be mapped to a RequestMessage and ResponseMessage. It is used to support
// both hiveot and WoT websocket message formats.
//
// Users must use ConnectWithToken to authenticate and connect.
//
//	wssURL is the full websocket connection URL
//	clientID is the authentication ID of the consumer or agent
//	caCert is the server CA for TLS connection validation
//	sink is the application module receiving notifications or in case of agents, requests.
//	timeout is the maximum connection wait time
func NewWotWssClient(
	wssURL string, caCert *x509.Certificate) *WssClient {

	timeout := transports.DefaultRpcTimeout

	urlParts, _ := url.Parse(wssURL)
	hostPort := urlParts.Host
	wssPath := urlParts.Path
	tlsClient := tlsclient.NewTLSClient(hostPort, nil, caCert, timeout)

	cl := &WssClient{
		caCert:               caCert,
		maxReconnectAttempts: 0,
		msgConverter:         converter.NewWotWssMsgConverter(),
		rnrChan:              msg.NewRnRChan(),
		tlsClient:            tlsClient,
		timeout:              timeout,
		wssPath:              wssPath,
	}
	var _ transports.IConnection = cl // interface check
	var _ modules.IHiveModule = cl    // interface check
	return cl
}

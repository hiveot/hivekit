package grpcclient

import (
	"crypto/x509"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/msg"
	"github.com/hiveot/hivekit/go/wot/td"
	jsoniter "github.com/json-iterator/go"
	"github.com/teris-io/shortid"
)

// gRPC transport client for hiveot
// This implements the ITransportClient interface
type GrpcTransportClient struct {
	modules.HiveModuleBase

	connectURL string
	caCert     *x509.Certificate
	clientID   string
	bufStream  *BufferedStream

	grpcClient *GrpcServiceClient

	// handler for sending connection notifications
	connectHandler transports.ConnectionHandler

	retryOnDisconnect atomic.Bool

	// the request & response channel handler
	// all responses are passed here to support response callbacks
	rnrChan *msg.RnRChan

	// send/receive timeout to use
	timeout time.Duration
}

// socket connection status handler
func (cl *GrpcTransportClient) _onConnectionChanged(connected bool, err error) {

	if cl.connectHandler != nil {
		cl.connectHandler(connected, cl, err)
	}
	// TODO: if retrying is enabled then try on disconnect
	if !connected && cl.retryOnDisconnect.Load() {
		// go cl.grpcClient.Connect()
	}
}

// Authenticate and connect
func (cl *GrpcTransportClient) Authenticate(tdDoc *td.TD,
	getCredentials transports.GetCredentials) error {
	return fmt.Errorf("not yet implemented")
}

// Close disconnects
func (cl *GrpcTransportClient) Close() {
	if cl.grpcClient != nil {
		cl.grpcClient.Close()
		cl.grpcClient = nil
	}
	if cl.bufStream != nil {
		cl.bufStream.Close()
		cl.bufStream = nil
	}
}

// ConnectWithToken attempts to establish a UDS connection
// clientID and token are not used.
func (cl *GrpcTransportClient) ConnectWithToken(clientID string, token string) (err error) {
	cl.clientID = clientID

	cl.grpcClient = NewGrpcServiceClient(
		clientID, cl.connectURL, cl.caCert, cl.timeout, cl._onMessage)
	err = cl.grpcClient.Connect()
	if err != nil {
		slog.Error("Grpc connection failed", "addr", cl.connectURL)
	}
	go func() {
		cl._onConnectionChanged(cl.grpcClient.IsConnected(), nil)
		cl.grpcClient.WaitUntilDisconnect()
		cl._onConnectionChanged(cl.grpcClient.IsConnected(), nil)
	}()
	return err
}

// GetClientID returns the client's connection details
func (cl *GrpcTransportClient) GetClientID() string {
	return cl.clientID
}

// // GetConnectionID returns the client's connection details
func (cl *GrpcTransportClient) GetConnectionID() string {
	return "todo need metadata"
}

// HandleNotification forwards notifications to the server instead of forwarding to their sink.
// incoming notifications are forwarded to the sink.
func (cl *GrpcTransportClient) HandleNotification(notif *msg.NotificationMessage) {
	cl.SendNotification(notif)
}

// HandleRequest forwards requests to the server
func (cl *GrpcTransportClient) HandleRequest(request *msg.RequestMessage, replyTo msg.ResponseHandler) error {
	err := cl.SendRequest(request, replyTo)
	return err
}

// onMessage processes the incoming message received from the server.
// This decodes the message into a request or response message and passes
// it to the application handler.
func (cl *GrpcTransportClient) _onMessage(messageType string, jsonRaw string) {

	switch messageType {
	case msg.MessageTypeNotification:
		// client consumer receives a notification
		var notif *msg.NotificationMessage
		err := jsoniter.UnmarshalFromString(jsonRaw, &notif)
		if err != nil {
			slog.Error("_onMessage: unmarshalling notification failed", "err", err.Error())
			return
		}
		go func() {
			cl.HiveModuleBase.HandleNotification(notif)
		}()

	case msg.MessageTypeRequest:
		// client agent receives a request (using reverse connection)
		go func() {
			var req *msg.RequestMessage
			err := jsoniter.UnmarshalFromString(jsonRaw, &req)
			if err != nil {
				slog.Error("_onMessage: unmarshalling request failed", "err", err.Error())
				return
			}
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

		}()

	case msg.MessageTypeResponse:
		// client consumer receives a response
		go func() {
			var resp *msg.ResponseMessage
			err := jsoniter.UnmarshalFromString(jsonRaw, &resp)
			if err != nil {
				slog.Error("_onMessage: unmarshalling response failed", "err", err.Error())
				return
			}
			// pass it on to the waiting consumer
			handled := cl.rnrChan.HandleResponse(resp, cl.timeout)
			if !handled {
				slog.Error("HandleWssMessage: received response but no matching request",
					"correlationID", resp.CorrelationID,
					"op", resp.Operation,
					"name", resp.Name,
					"clientID", cl.clientID,
				)
			}

		}()
	}
}

// IsConnected return whether the socket connection is established
func (cl *GrpcTransportClient) IsConnected() bool {
	return cl.bufStream.IsConnected()
}

// SendNotification Agent posts a notification to the server
func (cl *GrpcTransportClient) SendNotification(notif *msg.NotificationMessage) {
	clientID := cl.clientID
	slog.Info("SendNotification",
		slog.String("clientID", clientID),
		slog.String("correlationID", notif.CorrelationID),
		slog.String("affordance", string(notif.AffordanceType)),
		slog.String("thingID", notif.ThingID),
		slog.String("name", notif.Name),
	)
	notifJson, err := jsoniter.MarshalToString(notif)
	if err == nil {
		err = cl.bufStream.Send(msg.MessageTypeNotification, notifJson)
	}
}

// SendRequest send a request message the server
func (cl *GrpcTransportClient) SendRequest(
	req *msg.RequestMessage, replyTo msg.ResponseHandler) error {

	if req.CorrelationID == "" {
		req.CorrelationID = shortid.MustGenerate()
	}
	reqJson, _ := jsoniter.MarshalToString(req)

	if replyTo == nil {
		// responses are received asynchronously
		err := cl.bufStream.Send(msg.MessageTypeRequest, reqJson)
		return err
	}

	// a response handler is provided, callback when the response is received
	cl.rnrChan.Open(req.CorrelationID)
	err := cl.bufStream.Send(msg.MessageTypeRequest, reqJson)

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

// SendResponse send a response message to the server
func (cl *GrpcTransportClient) SendResponse(resp *msg.ResponseMessage) error {

	respJson, err := jsoniter.MarshalToString(resp)
	if err == nil {
		err = cl.bufStream.Send(msg.MessageTypeResponse, respJson)
	}
	return err
}

func (cl *GrpcTransportClient) SetTimeout(timeout time.Duration) {
	cl.timeout = timeout
}

// Module stop
func (cl *GrpcTransportClient) Stop() {
	cl.Close()
}

// NewGrpcTransportClient creates a new instance of the Hiveot UDS client
//
// when using network sockets, addr is the URL with CaCert the CA certificate to
// validate the server connection.
// Use SetTimeout to change the timeout for testing purposes.
//
// connectURL is the server URL, e.g.  unix://{/path.sock} or tcp://localhost:{port}
// caCert is the CA certificate to validate the server connection, or nil for UDS or insecure connections.
//
// Users must use ConnectWithToken to authenticate and start.
func NewGrpcTransportClient(connectURL string, caCert *x509.Certificate) *GrpcTransportClient {

	cl := &GrpcTransportClient{
		connectURL: connectURL,
		caCert:     caCert,
		timeout:    transports.DefaultRpcTimeout,
	}

	var _ transports.ITransportClient = cl // check interface implementation
	return cl
}

package grpcclient

import (
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/msg"
	"github.com/hiveot/hivekit/go/wot/td"
	jsoniter "github.com/json-iterator/go"
)

// gRPC client for hiveot
// This implements the ITransportClient interface
type GrpcTransportClient struct {
	modules.HiveModuleBase

	addr       string
	clientID   string
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
	// if retrying is enabled then try on disconnect
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
	}
}

// ConnectWithToken attempts to establish a UDS connection
// clientID and token are not used.
func (cl *GrpcTransportClient) ConnectWithToken(clientID string, token string) (err error) {
	cl.clientID = clientID

	cl.grpcClient = NewGrpcServiceClient(cl.addr, cl.timeout, cl._onMessage, cl._onConnectionChanged)
	err = cl.grpcClient.Connect()
	if err != nil {
		slog.Error("Grpc connection failed", "addr", cl.addr)
	}
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

// clients send requests to the server
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
	return cl.grpcClient.isConnected.Load()
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
	// convert the operation into a protocol message

	// err := cl.grpcClient.WriteMessage(notif)
	// if err != nil {
	// 	slog.Warn("SendNotification failed",
	// 		"clientID", clientID,
	// 		"err", err.Error())
	// }
}

// SendRequest send a request message the server
func (cl *GrpcTransportClient) SendRequest(
	req *msg.RequestMessage, replyTo msg.ResponseHandler) error {

	return fmt.Errorf("not yet implemented")
}

// SendResponse send a response message to the server
func (cl *GrpcTransportClient) SendResponse(resp *msg.ResponseMessage) error {
	return fmt.Errorf("not yet implemented")
}

func (cl *GrpcTransportClient) SetTimeout(timeout time.Duration) {
	cl.timeout = timeout
}

// NewHiveotGrpcClient creates a new instance of the Hiveot UDS client
func NewHiveotGrpcClient(addr string, timeout time.Duration) *GrpcTransportClient {

	cl := &GrpcTransportClient{
		timeout: timeout,
		addr:    addr,
	}
	// cl.grpcClient = NewGrpcClient(
	// socketPath, timeout, cl.HandleUdsMessage, cl._onConnectionChanged)

	var _ transports.ITransportClient = cl
	return cl
}

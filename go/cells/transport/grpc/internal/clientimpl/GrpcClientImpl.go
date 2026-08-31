package clientimpl

import (
	"crypto/x509"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/cells/transport"
	grpctransport "github.com/hiveot/hivekit/go/cells/transport/grpc"
	"github.com/hiveot/hivekit/go/cells/transport/grpc/internal"
	"github.com/teris-io/shortid"
)

// gRPC transport client for hiveot
// This implements the ITransportClient interface
type GrpcClientImpl struct {
	*transport.TransportClientBase

	connectURL string
	rootCAs    *x509.CertPool

	// encoding and decoding of RRN messages
	encoder transport.IMessageEncoder

	// the underlying grpc client. Set on authenticate. nil when closed
	grpcSvcClient *internal.GrpcServiceClient

	// the request & response channel handler
	// all responses are passed here to support response callbacks
	rnrChan *msg.RnRChan
}

// _onGrpcClientMessage processes the incoming message received from the server.
// This decodes the message into a request or response message and passes
// it to the application handler.
func (cl *GrpcClientImpl) _onGrpcClientMessage(raw []byte) {
	var err error
	if raw == nil {
		slog.Error("_onGrpcClientMessage: raw data is nil")
	}
	msgType := cl.encoder.DetermineMessageType(raw)
	switch msgType {
	case msg.MessageTypeNotification:
		var notif *msg.NotificationMessage
		notif, err = cl.encoder.DecodeNotification("", raw)
		if err == nil {
			go func() {
				cl.HiveCellBase.HandleNotification(notif)
			}()
			return
		}
	case msg.MessageTypeRequest:
		var req *msg.RequestMessage
		req, err = cl.encoder.DecodeRequest("", raw)
		if err == nil {
			// client receives a request (device with reverse connection)
			go func() {
				// pass it on to the linked producer.
				err = cl.EmitRequest(req, func(resp *msg.ResponseMessage) error {
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
			return
		}
	case msg.MessageTypeResponse:
		var resp *msg.ResponseMessage
		resp, err = cl.encoder.DecodeResponse("", raw)
		if err == nil {
			// client consumer receives a response
			go func() {
				// pass it on to the waiting consumer
				handled := cl.rnrChan.HandleResponse(resp, cl.GetTimeout())
				if !handled {
					slog.Error("_onGrpcClientMessage: received response but no matching request",
						"correlationID", resp.CorrelationID,
						"op", resp.Operation,
						"name", resp.Name,
						"clientID", cl.GetClientID(),
					)
				}
			}()
			return
		}
	default:
		err = fmt.Errorf("Unknown message type '%s'", msgType)
	}
	slog.Error("_onGrpcClientMessage: Failed to handle message", "err", err.Error())
}

// update the connection status and publish an notification if it differs from the last status
// a 'lost' status is ignored if the current status is set to closed as it was intentional.
// a lost status cancels all waiting requests.
func (cl *GrpcClientImpl) _setConnectionStatus(newStatus api.ConnectionStatus, err error) {

	if newStatus == api.StatusLost {
		slog.Info("_setConnectionStatus SseCl client connection lost", "status", newStatus)
		// fail all outstanding RnR requests
		cl.rnrChan.CloseAll()
	}
	cl.TransportClientBase.SetConnectionStatus(newStatus, err)
}

// Close disconnects the current connection and publish a closed notification
func (cl *GrpcClientImpl) Close() {

	// set status to closed first to avoid a reconnect
	cl._setConnectionStatus(api.StatusClosed, nil)

	if cl.grpcSvcClient != nil {
		cl.grpcSvcClient.Close()
		cl.grpcSvcClient = nil
	}
}

// Connect attempts to establish the streams using the previously set authentication method
// If this fails due to credentials it returns UnauthorizedError
func (cl *GrpcClientImpl) Connect() (err error) {

	status := cl.GetConnectionStatus()
	if status == api.StatusConnected {
		return fmt.Errorf("Already connected")
	} else if status == api.StatusConnecting {
		return fmt.Errorf("Busy connecting")
	}

	// create the grpc client to use but do not connect yet
	clientCert := cl.GetClientCert()
	authToken, scheme := cl.GetAuthToken()
	_ = scheme
	clientID := cl.GetClientID()
	cl.grpcSvcClient = internal.StartGrpcServiceClient(
		cl.connectURL, clientID, authToken, clientCert, cl.rootCAs, cl.GetTimeout(),
		grpctransport.GrpcTransportServiceName, cl._onGrpcClientMessage)

	// new connect attempt
	cl._setConnectionStatus(api.StatusConnecting, nil)
	err = cl.grpcSvcClient.Connect()

	// use ping as 'connect' might not detect a failed connection
	_, err = cl.grpcSvcClient.Ping("")
	if err != nil {
		slog.Error(err.Error(), "url", cl.connectURL)
		cl._setConnectionStatus(api.StatusLost, err)
		return err
	}

	// connect the streams want serve
	_, err = cl.grpcSvcClient.ConnectStream(grpctransport.StreamNameNotification)
	if err == nil {
		// FIXME: make dual stream work
		// _, err = cl.grpcClient.ConnectStream(grpcapi.StreamNameRequestResponse)
	}

	go func() {
		// for now assume that the notification stream drives the connectivity.
		// the req/resp stream should follow like a good doggie
		name := grpctransport.StreamNameNotification
		if cl.grpcSvcClient.IsConnected(name) {
			cl._setConnectionStatus(api.StatusConnected, nil)

			cl.grpcSvcClient.WaitUntilDisconnect(name)
			cl._setConnectionStatus(api.StatusLost, nil)

		} else {
			slog.Error("Connect: connection unexpectedly dropped")
		}
	}()

	// allow background tasks to complete
	time.Sleep(time.Millisecond)

	return nil
}

// HandleNotification sends notifications to the server.
// Incoming notifications are emitted to the sink.
func (cl *GrpcClientImpl) HandleNotification(notif *msg.NotificationMessage) {
	cl.SendNotification(notif)
}

// Clients receives a request
// - reconnect actions are handled here
// - other requests (like subscribe) are send to the server
func (cl *GrpcClientImpl) HandleRequest(request *msg.RequestMessage, replyTo msg.ResponseHandler) error {
	if request.ThingID == cl.GetConnectionID() {
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

// SendNotification exposed thing posts a notification to the server
func (cl *GrpcClientImpl) SendNotification(notif *msg.NotificationMessage) {
	if cl.GetConnectionStatus() != api.StatusConnected {
		slog.Error("SendNotification: Not connected")
	}

	clientID := cl.GetClientID()
	slog.Info("SendNotification",
		slog.String("clientID", clientID),
		slog.String("correlationID", notif.CorrelationID),
		slog.String("affordance", string(notif.AffordanceType)),
		slog.String("thingID", notif.ThingID),
		slog.String("name", notif.Name),
	)
	raw, err := cl.encoder.EncodeNotification(notif)
	if err == nil {
		err = cl.grpcSvcClient.Send(grpctransport.StreamNameNotification, raw)
	}
}

// SendRequest send a request message the server
func (cl *GrpcClientImpl) SendRequest(
	req *msg.RequestMessage, replyTo msg.ResponseHandler) error {

	if cl.GetConnectionStatus() != api.StatusConnected {
		return fmt.Errorf("SendRequest: Not connected")
	}

	if req.CorrelationID == "" {
		req.CorrelationID = shortid.MustGenerate()
	}
	raw, err := cl.encoder.EncodeRequest(req)
	if err != nil {
		slog.Error("SendRequest: unknown request", "op", req.Operation, "err", err.Error())
		return err
	}
	if replyTo == nil {
		// responses are received asynchronously
		err := cl.grpcSvcClient.Send(grpctransport.StreamNameNotification, raw)
		// FIXME: make dual stream work
		// err := cl.grpcClient.Send(grpcapi.StreamNameRequestResponse, raw)
		return err
	}

	// a response handler is provided, callback when the response is received
	cl.rnrChan.Open(req.CorrelationID)
	err = cl.grpcSvcClient.Send(grpctransport.StreamNameNotification, raw)
	// FIXME: make dual stream work
	// err = cl.grpcClient.Send(grpcapi.StreamNameRequestResponse, raw)

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

// SendResponse send a response message to the server
func (cl *GrpcClientImpl) SendResponse(resp *msg.ResponseMessage) error {

	if cl.GetConnectionStatus() != api.StatusConnected {
		return fmt.Errorf("SendResponse: Not connected")
	}

	raw, err := cl.encoder.EncodeResponse(resp)
	if err == nil {
		// FIXME: make dual stream work
		err = cl.grpcSvcClient.Send(grpctransport.StreamNameNotification, raw)
		// err = cl.grpcClient.Send(grpcapi.StreamNameRequestResponse, raw)
	}
	return err
}

// Start the transport client and attempt to connect to the server if not already connected.
//
// Intended for use by the factory as the factory provides a clientID/token or client
// certificate. This just calls Connect().
// func (cl *GrpcClientImpl) Start() error {
// 	err := cl.Connect()
// 	return err
// }

// Stop the client instance
func (cl *GrpcClientImpl) Stop() {
	cl.Close()
}

// StartGrpcClientImpl creates a new instance of the Hiveot gRPC client.
//
// To use, authenticate and call Connect.
//
// Note that go-gRPC uses the 'dns' scheme and does not support 'tcp'. In order
// to remain consistent with the server, this client maps the 'tcp' scheme to 'dns'
// when needed.
// The ipv4 scheme is not supported.
//
// Use SetTimeout to change the timeout for testing purposes.
//
// connectURL is the server URL, e.g.  unix://{/path.sock}, tcp://localhost:{port} or simply "address:port"
// rootCAs contains the CA certificates to validate the server connection, or nil for UDS or insecure connections.
// ch is the connect/disconnect callback
func StartGrpcClientImpl(connectURL string, rootCAs *x509.CertPool) *GrpcClientImpl {

	// gRPC does not support tcp scheme, but we want to allow users to specify it for consistency with the server.
	connectURL = strings.TrimPrefix(connectURL, "tcp://")
	thingID := "grpc-client-" + shortid.MustGenerate()
	timeout := msg.DefaultRnRTimeout

	cl := &GrpcClientImpl{
		TransportClientBase: transport.NewTransportClientBase(thingID, rootCAs, timeout),
		rootCAs:             rootCAs,
		connectURL:          connectURL,
		encoder:             transport.NewRRNJsonEncoder(),
		rnrChan:             msg.NewRnRChan(),
	}

	var _ api.ITransportClient = cl // check interface implementation
	return cl
}

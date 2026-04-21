package grpcclient

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"strings"
	"sync/atomic"
	"time"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/transports"
	grpctransport "github.com/hiveot/hivekit/go/modules/transports/grpc"
	grpclib "github.com/hiveot/hivekit/go/modules/transports/grpc/lib"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/teris-io/shortid"
)

// gRPC transport client for hiveot
// This implements the ITransportClient interface
type GrpcTransportClient struct {
	modules.HiveModuleBase

	bearerToken string

	// handler for sending connection notifications
	connectHandler transports.ConnectionHandler

	connectURL string
	caCert     *x509.Certificate
	clientCert *tls.Certificate
	clientID   string

	// encoding and decoding of RRN messages
	encoder transports.IMessageEncoder

	grpcClient *grpclib.GrpcServiceClient

	maxReconnectAttempts int // 0 for indefinite

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
}

// onMessage processes the incoming message received from the server.
// This decodes the message into a request or response message and passes
// it to the application handler.
func (cl *GrpcTransportClient) _onClientMessage(raw []byte) {

	if raw == nil {
		slog.Error("_onClientMessage: raw data is nil")
	}
	// assume message is a notification (most likely)
	notif, err := cl.encoder.DecodeNotification(raw)
	if err == nil {
		go func() {
			cl.HiveModuleBase.HandleNotification(notif)
		}()
		return
	}
	// assume message is request
	req, err := cl.encoder.DecodeRequest(raw)
	if err == nil {
		// client agent receives a request (using reverse connection)
		go func() {
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
		return
	}
	// remaining option
	resp, err := cl.encoder.DecodeResponse(raw)
	if err == nil {
		// client consumer receives a response
		go func() {
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
		return
	}

	slog.Error("_onClientMessage: Failed to unmarshal message", "err", err.Error())
}

// Authenticate and connect
func (cl *GrpcTransportClient) Authenticate(tdDoc *td.TD,
	getCredentials transports.GetCredentials) error {
	return fmt.Errorf("not yet implemented")
}

// Close disconnects
func (cl *GrpcTransportClient) Close() {
	// dont try to reconnect after close
	cl.retryOnDisconnect.Store(false)

	if cl.grpcClient != nil {
		cl.grpcClient.Close()
	}
}

// ConnectWithToken attempts to establish a UDS connection
// clientID and token are not used.
func (cl *GrpcTransportClient) ConnectWithToken(clientID string, token string) (err error) {

	// ensure disconnected (note that this resets retryOnDisconnect)
	if cl.IsConnected() {
		cl.Close()
	}

	cl.clientID = clientID
	cl.bearerToken = token

	cl.grpcClient = grpclib.NewGrpcServiceClient(
		cl.connectURL, cl.clientCert, cl.caCert, cl.timeout,
		grpctransport.GrpcTransportServiceName, cl._onClientMessage)

	err = cl.grpcClient.ConnectWithToken(clientID, token)
	if err != nil {
		slog.Error("Grpc connection failed", "addr", cl.connectURL, "err", err.Error())
		return err
	}

	// use ping as 'connect' might not detect a failed connection
	_, err = cl.grpcClient.Ping("")
	if err != nil {
		slog.Error(err.Error(), "url", cl.connectURL)
		return err
	}

	// connect the streams want serve
	_, err = cl.grpcClient.ConnectStream(grpctransport.StreamNameNotification)
	if err == nil {
		// FIXME: make dual stream work
		// _, err = cl.grpcClient.ConnectStream(grpcapi.StreamNameRequestResponse)
	}

	go func() {
		// for now assume that the notification stream drives the connectivity.
		// the req/resp stream should follow like a good doggie
		name := grpctransport.StreamNameNotification
		if cl.grpcClient.IsConnected(name) {
			cl._onConnectionChanged(cl.grpcClient.IsConnected(name), nil)
			cl.grpcClient.WaitUntilDisconnect(name)
			cl._onConnectionChanged(cl.grpcClient.IsConnected(name), nil)

			// if retrying is enabled then try on disconnect
			if cl.retryOnDisconnect.Load() {
				go cl.Reconnect()
			}
		} else {
			slog.Error("ConnectWithToken: connection unexpectedly dropped")
		}
	}()

	// even if connection failed right now, enable retry
	cl.retryOnDisconnect.Store(true)

	// allow background tasks to complete
	time.Sleep(time.Millisecond)

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

// IsConnected return whether the notification stream is established
func (cl *GrpcTransportClient) IsConnected() bool {
	return cl.grpcClient != nil && cl.grpcClient.IsConnected(grpctransport.StreamNameNotification)
}

// Reconnect attempts to re-establish a dropped connection using the last token
// This uses an increasing backoff period up to 15 seconds, starting random between 0-2 seconds
func (cl *GrpcTransportClient) Reconnect() {
	var err error
	var backoffDuration time.Duration = time.Duration(rand.Uint64N(uint64(time.Second * 2)))

	for i := 0; cl.maxReconnectAttempts == 0 || i < cl.maxReconnectAttempts; i++ {
		// retry until max repeat is reached, disconnect is called or authorization failed
		if !cl.retryOnDisconnect.Load() {
			break
		}
		slog.Warn("gRPC Reconnecting attempt",
			slog.String("clientID", cl.clientID),
			slog.Int("i", i))
		err = cl.ConnectWithToken(cl.clientID, cl.bearerToken)
		if err == nil {
			break
		}
		if errors.Is(err, utils.UnauthorizedError) {
			break
		}
		// the connection timeout doesn't seem to work for some reason
		//
		time.Sleep(backoffDuration)
		// slowly wait longer until 15 sec.
		if backoffDuration < time.Second*15 {
			backoffDuration += time.Second
		}
	}
	if err != nil {
		slog.Warn("Reconnect failed: ", "err", err.Error())
	}
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
	raw, err := cl.encoder.EncodeNotification(notif)
	if err == nil {
		err = cl.grpcClient.Send(grpctransport.StreamNameNotification, raw)
	}
}

// SendRequest send a request message the server
func (cl *GrpcTransportClient) SendRequest(
	req *msg.RequestMessage, replyTo msg.ResponseHandler) error {

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
		err := cl.grpcClient.Send(grpctransport.StreamNameNotification, raw)
		// FIXME: make dual stream work
		// err := cl.grpcClient.Send(grpcapi.StreamNameRequestResponse, raw)
		return err
	}

	// a response handler is provided, callback when the response is received
	cl.rnrChan.Open(req.CorrelationID)
	err = cl.grpcClient.Send(grpctransport.StreamNameNotification, raw)
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
	hasResponse, resp := cl.rnrChan.WaitForResponse(req.CorrelationID, cl.timeout)
	if hasResponse {
		err = replyTo(resp)
	}
	return err
}

// SendResponse send a response message to the server
func (cl *GrpcTransportClient) SendResponse(resp *msg.ResponseMessage) error {

	raw, err := cl.encoder.EncodeResponse(resp)
	if err == nil {
		// FIXME: make dual stream work
		err = cl.grpcClient.Send(grpctransport.StreamNameNotification, raw)
		// err = cl.grpcClient.Send(grpcapi.StreamNameRequestResponse, raw)
	}
	return err
}

func (cl *GrpcTransportClient) SetTimeout(timeout time.Duration) {
	cl.timeout = timeout
}

// Start the module. Use ConnectWithToken instead
func (cl *GrpcTransportClient) Start() error {
	return nil
}

// Module stop
func (cl *GrpcTransportClient) Stop() {
	cl.Close()
}

// NewGrpcTransportClient creates a new instance of the Hiveot gRPC client
//
// Note that go-gRPC uses the 'dns' scheme and does not support 'tcp'. In order
// to remain consistent with the server, this client maps the 'tcp' scheme to 'dns'
// when needed.
// The ipv4 scheme is not supported.
//
// Use SetTimeout to change the timeout for testing purposes.
//
// connectURL is the server URL, e.g.  unix://{/path.sock}, tcp://localhost:{port} or simply "address:port"
// caCert is the CA certificate to validate the server connection, or nil for UDS or insecure connections.
// ch is the connect/disconnect callback
//
// Users must use ConnectWithToken to authenticate and start.
func NewGrpcTransportClient(
	connectURL string, clientCert *tls.Certificate, caCert *x509.Certificate,
	ch transports.ConnectionHandler) *GrpcTransportClient {

	// gRPC does not support tcp scheme, but we want to allow users to specify it for consistency with the server.
	connectURL = strings.TrimPrefix(connectURL, "tcp://")

	cl := &GrpcTransportClient{
		caCert:               caCert,
		clientCert:           clientCert,
		connectHandler:       ch,
		connectURL:           connectURL,
		encoder:              transports.NewRRNJsonEncoder(),
		maxReconnectAttempts: 0,
		rnrChan:              msg.NewRnRChan(),
		timeout:              msg.DefaultRnRTimeout,
	}

	var _ transports.ITransportClient = cl // check interface implementation
	return cl
}

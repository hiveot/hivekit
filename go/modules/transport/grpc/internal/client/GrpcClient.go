package internalclient

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/transport"
	grpctransport "github.com/hiveot/hivekit/go/modules/transport/grpc"
	"github.com/hiveot/hivekit/go/modules/transport/grpc/internal"
	"github.com/teris-io/shortid"
)

// gRPC transport client for hiveot
// This implements the ITransportClient interface
type GrpcClient struct {
	*modules.HiveModuleBase

	bearerToken string

	// status of current connection
	connectStatus api.ConnectionStatus
	// callback when connection changes
	connectHandler func(newStatus api.ConnectionStatus, c api.ITransportClient)

	// instance ID used to identify the client and its connection
	connectionID string

	connectURL string
	caCert     *x509.Certificate
	clientCert *tls.Certificate
	clientID   string

	// encoding and decoding of RRN messages
	encoder transport.IMessageEncoder

	// Close is called so a disconnect is expected
	// isClosed atomic.Bool

	// the underlying grpc client. Set on authenticate. nil when closed
	grpcSvcClient *internal.GrpcServiceClient

	// variables access
	mux sync.RWMutex

	// the request & response channel handler
	// all responses are passed here to support response callbacks
	rnrChan *msg.RnRChan
}

// // socket connection status handler
// // This emits a notification if the connection is established, lost or disconnected.
// func (cl *GrpcClient) _onConnectionChanged(newStatus api.ConnectionStatus, err error) {

// 	// 1. update the connection status
// 	cl.mux.Lock()
// 	cl.connectStatus = newStatus
// 	cl.mux.Unlock()
// 	cl._setConnectionStatus(newStatus, err)
// }

// _onClientMessage processes the incoming message received from the server.
// This decodes the message into a request or response message and passes
// it to the application handler.
func (cl *GrpcClient) _onClientMessage(raw []byte) {

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
		// client receives a request (using reverse connection)
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
			handled := cl.rnrChan.HandleResponse(resp, cl.GetTimeout())
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

// update the connection status and publish an notification if it differs from the last status
// a 'lost' status is ignored if the current status is set to closed as it was intentional.
// a lost status cancels all waiting requests.
func (cl *GrpcClient) _setConnectionStatus(
	newStatus api.ConnectionStatus, err error) {

	cl.mux.RLock()
	oldStatus := cl.connectStatus
	cl.mux.RUnlock()

	if newStatus == oldStatus {
		return
	} else if oldStatus == api.StatusClosed && newStatus == api.StatusLost {
		return
	} else if newStatus == api.StatusLost {
		slog.Info("_setConnectionStatus gRPC client connection lost", "status", newStatus)
		// fail all outstanding RnR requests
		cl.rnrChan.CloseAll()
	}
	cl.mux.Lock()
	cl.connectStatus = newStatus
	ch := cl.connectHandler
	cl.mux.Unlock()

	// notify upstream of status change. the cid is the client instance thingID
	cid := cl.GetConnectionID()
	evName := api.ClientConnectionStatusEvent
	notif := msg.NewNotificationMessage(
		cid, msg.AffordanceTypeEvent, cid, evName, newStatus)
	cl.ForwardNotification(notif)

	// invoke the callback after the notification so that the proper sequence is maintained
	// if the callback tries to reconnect.
	if ch != nil {
		ch(newStatus, cl)
	}
}

// AuthenticateWithClientCert sets the authentication credentials to the client certificate.
func (cl *GrpcClient) AuthenticateWithClientCert(clientCert *tls.Certificate) (err error) {
	status := cl.GetConnectionStatus()
	if status == api.StatusConnected || status == api.StatusConnecting {
		return fmt.Errorf("AuthenticateWithClientCert: Connection in progress.")
	}

	// verify the validity of this certificate against the CA
	// without this one can spend a long time figuring out why the connection fails.
	x509Cert, err := x509.ParseCertificate(clientCert.Certificate[0])
	if err == nil {
		// cert subject is clientID
		cl.clientID = x509Cert.Subject.CommonName
		caCertPool := x509.NewCertPool()
		caCertPool.AddCert(cl.caCert)
		opts := x509.VerifyOptions{
			Roots:     caCertPool,
			KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		}
		_, err = x509Cert.Verify(opts)
	}
	if err != nil {
		slog.Error("AuthenticateWithClientCert failed: " + err.Error())
		return err
	}
	cl.clientCert = clientCert

	// create the grpc client to use but do not connect yet
	cl.grpcSvcClient = internal.NewGrpcServiceClient(
		cl.connectURL, clientCert, cl.caCert, cl.GetTimeout(),
		grpctransport.GrpcTransportServiceName, cl._onClientMessage)

	return err
}

// Authenticate
func (cl *GrpcClient) AuthenticateWithForm(
	tdDoc *td.TD, getCredentials api.GetCredentials) error {

	status := cl.GetConnectionStatus()
	if status == api.StatusConnected || status == api.StatusConnecting {
		return fmt.Errorf("AuthenticateWithForm: Connection already in progress.")
	}
	clientID, secret, schemeName, err := getCredentials(tdDoc.ID)

	secScheme, err := tdDoc.GetSecurityScheme()
	if secScheme.Scheme == td.SecSchemeNoSec {
		// a unix socket relies on the filesystem permissions
	} else if schemeName != secScheme.Scheme && schemeName != "" && schemeName != td.SecSchemeAuto {
		err = fmt.Errorf("AuthenticateWithForm: TD Security scheme doesn't match credentials TD scheme='%s', credentials scheme='%s'", secScheme.Scheme, schemeName)
	} else if secScheme.Scheme == td.SecSchemeBearer || secScheme.Scheme == td.SecSchemeAuto {
		err = cl.AuthenticateWithToken(clientID, secret)
	} else {
		err = fmt.Errorf("AuthenticateWithForm: Unsupported security scheme '%s'", secScheme.Scheme)
	}
	return err
}

// AuthenticateWithToken sets the token credentials to use in Connect
// and create the underlying grpc service client with the credentials to use.
func (cl *GrpcClient) AuthenticateWithToken(clientID string, token string) (err error) {

	status := cl.GetConnectionStatus()
	if status == api.StatusConnected || status == api.StatusConnecting {
		return fmt.Errorf("AuthenticateWithToken: Connection in progress.")
	}

	cl.clientID = clientID
	cl.bearerToken = token

	// create the grpc client to use but do not connect yet
	cl.grpcSvcClient = internal.NewGrpcServiceClient(
		cl.connectURL, cl.clientCert, cl.caCert, cl.GetTimeout(),
		grpctransport.GrpcTransportServiceName, cl._onClientMessage)

	err = cl.grpcSvcClient.AuthenticateWithToken(clientID, token)
	return err
}

// Close disconnects the current connection and publish a closed notification
func (cl *GrpcClient) Close() {

	// set status to closed first to avoid a reconnect
	cl._setConnectionStatus(api.StatusClosed, nil)

	cl.mux.Lock()
	defer cl.mux.Unlock()
	if cl.grpcSvcClient != nil {
		cl.grpcSvcClient.Close()
		cl.grpcSvcClient = nil
	}
}

// Connect attempts to establish the streams using the previously set authentication method
func (cl *GrpcClient) Connect() (err error) {

	status := cl.GetConnectionStatus()
	if cl.grpcSvcClient == nil {
		return fmt.Errorf("Auth credentials not set")
	} else if status == api.StatusConnected {
		return nil
	} else if status == api.StatusConnecting {
		return fmt.Errorf("Busy connecting")
	}

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
			slog.Error("AuthenticateWithToken: connection unexpectedly dropped")
		}
	}()

	// allow background tasks to complete
	time.Sleep(time.Millisecond)

	return nil
}

// GetClientID returns the client's connection details
func (cl *GrpcClient) GetClientID() string {
	return cl.clientID
}

// GetConnectionID returns the client's instance-ID
// the connectionID is also used as the client's ThingID
func (cl *GrpcClient) GetConnectionID() string {
	return cl.connectionID
}

// // GetConnectionStatus returns the current connection status
func (cl *GrpcClient) GetConnectionStatus() api.ConnectionStatus {
	cl.mux.RLock()
	defer cl.mux.RUnlock()
	stat := cl.connectStatus
	return stat
}

// HandleNotification forwards notifications to the server instead of forwarding to their sink.
// incoming notifications are forwarded to the sink.
func (cl *GrpcClient) HandleNotification(notif *msg.NotificationMessage) {
	cl.SendNotification(notif)
}

// Clients receives a request
// - reconnect actions are handled here
// - other requests (like subscribe) are send to the server
func (cl *GrpcClient) HandleRequest(request *msg.RequestMessage, replyTo msg.ResponseHandler) error {
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

// // IsConnected return whether the notification stream is established
// func (cl *GrpcClient) IsConnected() bool {
// 	return cl.grpcSvcClient != nil && cl.grpcSvcClient.IsConnected(grpctransport.StreamNameNotification)
// }

// SendNotification exposed thing posts a notification to the server
func (cl *GrpcClient) SendNotification(notif *msg.NotificationMessage) {
	if cl.GetConnectionStatus() != api.StatusConnected {
		slog.Error("SendNotification: Not connected")
	}

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
		err = cl.grpcSvcClient.Send(grpctransport.StreamNameNotification, raw)
	}
}

// SendRequest send a request message the server
func (cl *GrpcClient) SendRequest(
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
func (cl *GrpcClient) SendResponse(resp *msg.ResponseMessage) error {

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

// SetConnectHandler sets the callback to invoke when the connection status changes
func (cl *GrpcClient) SetConnectHandler(
	h func(newStatus api.ConnectionStatus, c api.ITransportClient)) {
	cl.mux.Lock()
	defer cl.mux.Unlock()
	cl.connectHandler = h
}

// Start the module and attempt to connect to the server if not already connected.
//
// Intended for use by the factory as the factory provides a clientID/token or client
// certificate.
//
// Most users will use AuthenticateWithToken() followed by Connect() instead.
func (cl *GrpcClient) Start() error {
	err := cl.Connect()
	return err
}

// Module stop
func (cl *GrpcClient) Stop() {
	cl.Close()
}

// NewGrpcClient creates a new instance of the Hiveot gRPC client.
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
// Users must use AuthenticateWithToken to authenticate and start.
func NewGrpcClient(
	connectURL string, caCert *x509.Certificate) *GrpcClient {

	// gRPC does not support tcp scheme, but we want to allow users to specify it for consistency with the server.
	connectURL = strings.TrimPrefix(connectURL, "tcp://")
	thingID := "grpc-client-" + shortid.MustGenerate()

	cl := &GrpcClient{
		HiveModuleBase: modules.NewHiveModuleBase(thingID, msg.DefaultRnRTimeout),
		caCert:         caCert,
		clientCert:     nil,
		connectionID:   shortid.MustGenerate(),
		connectURL:     connectURL,
		encoder:        transport.NewRRNJsonEncoder(),
		rnrChan:        msg.NewRnRChan(),
	}

	var _ api.ITransportClient = cl // check interface implementation
	return cl
}

package grpcserver

import (
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/msg"
	"google.golang.org/grpc"
)

const DefaultUDSModuleID = "hiveot-uds"

// GrpcTransportServer is the transport server using gRPC connections.
//
// This implements both ITransportServer and IHiveModule interfaces.
// The embedded TransportServerBase is used for managing connections and forwarding messages to sinks.
type GrpcTransportServer struct {
	transports.TransportServerBase
	// Authenticate
	authn transports.IAuthenticator

	tlsCert *tls.Certificate

	grpcService *GrpcServiceServer

	connectURL string
	// grpcServer *grpc.Server

	respTimeout time.Duration
}

// a request passed to this server is forwarded to the connection with the matching ID
// this is intended for passing requests to agents that have a reverse connection.
func (m *GrpcTransportServer) HandleRequest(
	req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {

	// first attempt to procss the when targeted at this module
	if req.ThingID == m.GetModuleID() {
		// currently nothing to do here
		// err = m.msgAPI.HandleRequest(req, replyTo)
		err = fmt.Errorf("HandleRequest: no operations for this module '%s' re defined: ", m.GetModuleID())
	} else {
		err = m.TransportServerBase.HandleRequest(req, replyTo)
	}
	return err
}

// The grpc service callback handler for incoming stream connections.
// This creates a new transport connection for the stream and blocks until the stream is closed.
func (m *GrpcTransportServer) ServeStreamConnection(clientID string, cid string, grpcStream grpc.ServerStream) error {

	// authentication???

	// Create a hiveot transport connection for this stream.
	c := StartGrpcTransportConnection(clientID, cid, grpcStream,
		m.ForwardRequest, m.ForwardNotification)
	c.SetTimeout(m.respTimeout)

	m.AddConnection(c)
	// must block until connection closes
	c.WaitUntilDisconnect()
	m.RemoveConnection(c)
	return nil
}

func (m *GrpcTransportServer) Start(yamlConfig string) (err error) {
	// FIXME: use the URL scheme to support network tcp and unix sockets
	// connectURL := fmt.Sprintf("unix://%s", m.udsPath)
	m.Init(DefaultUDSModuleID,
		transports.ProtocolTypeHiveotGrpc,
		transports.SubprotocolHiveotGrpc,
		m.connectURL, m.authn)

	udsFilePath := m.connectURL
	// start listening on unix sockets. Make sure the directory exists and the socket file doesn't.
	udsFilePath = strings.TrimPrefix(udsFilePath, "uds://")
	udsFilePath = strings.TrimPrefix(udsFilePath, "unix://")
	udsDir := filepath.Dir(udsFilePath)
	err = os.MkdirAll(udsDir, 0700)
	err = os.RemoveAll(udsFilePath)

	lis, err := net.Listen("unix", udsFilePath)
	if err != nil {
		return err
	}
	grpcAuthn := NewGrpcAuthenticator(m.authn)
	m.grpcService, err = StartGrpcServiceServer(
		lis, nil, m.ServeStreamConnection, grpcAuthn, time.Minute)
	if err != nil {
		lis.Close()
		return err
	}

	return err
}

// Stop any running actions
func (m *GrpcTransportServer) Stop() {
	slog.Info("Stop: Stopping gRPC module")
	m.CloseAll()
}

// GRPC server using UDS or TCP sockets
//
// connectURL is the URL to listen on, e.g. unix://{/path.sock} or tcp://localhost:{port}
// tlsCert is the TLS certificate to use for secure connections, or nil for insecure
// authn is the authenticator for verifying the client token
// respTimeout is the time the server waits for a response when sending requests. defaults to 3sec
func NewHiveotGrpcServer(connectURL string, tlsCert *tls.Certificate, authn transports.IAuthenticator, respTimeout time.Duration) *GrpcTransportServer {

	srv := &GrpcTransportServer{
		authn:       authn,
		connectURL:  connectURL,
		tlsCert:     tlsCert,
		respTimeout: respTimeout,
	}
	return srv
}

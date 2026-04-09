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

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/modules/transports"
	grpcapi "github.com/hiveot/hivekit/go/modules/transports/grpc/api"
	grpclib "github.com/hiveot/hivekit/go/modules/transports/grpc/lib"
	"github.com/hiveot/hivekit/go/utils"
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

	grpcService *grpclib.GrpcServiceServer

	connectURL string
	// grpcServer *grpc.Server

	respTimeout time.Duration

	// the service name the streams are published under
	serviceName string
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
func (m *GrpcTransportServer) ServeStreamConnection(
	clientID string, cid string, grpcStream grpc.ServerStream) error {

	// authentication???

	// Create a hiveot transport connection for this stream.
	c := StartGrpcTransportConnection(clientID, cid, grpcStream, m.ForwardRequest, m.ForwardNotification)
	c.SetTimeout(m.respTimeout)

	m.AddConnection(c)
	// must block until connection closes
	c.WaitUntilDisconnect()
	m.RemoveConnection(c)
	return nil
}

// Start the server with the given configuration.
// The server will listen on the configured URL and handle incoming connections.
// This adapts the URL scheme "unix", "uds", or "tcp" to the appropriate network type for net.Listen
// and update the connectURL to match the scheme used for listening.
func (m *GrpcTransportServer) Start(yamlConfig string) (err error) {

	address := m.connectURL
	network := "tcp"
	// start listening on unix sockets. Make sure the directory exists and the socket file doesn't.
	if strings.HasPrefix(address, "unix") {
		// m.connectURL is the same for the client
		network = "unix"
		address = strings.TrimPrefix(address, "unix://")
		socketDir := filepath.Dir(address)
		err = os.Remove(address)
		err = os.MkdirAll(socketDir, 0700)
	} else if strings.HasPrefix(address, "dns") {
		// m.connectURL is the same for the client
		address = strings.TrimPrefix(address, "dns:///") // dns scheme use triple slashes
		network = "tcp"
	} else if strings.HasPrefix(address, "tcp") {
		// gRPC clients do not support tcp scheme. remove it and use the server IP
		address = strings.TrimPrefix(address, "tcp://")
		network = "tcp"
	} else {
		// some unknown or missing scheme. Use the tcp scheme instead.
		network = "tcp"
	}
	if strings.HasPrefix(address, ":") {
		port := address
		outboundIP := utils.GetOutboundIP("")
		m.connectURL = fmt.Sprintf("tcp://%s%s", outboundIP.String(), port)
	} else {
		// full address to connect
		m.connectURL = fmt.Sprintf("%s://%s", network, address)
	}

	lis, err := net.Listen(network, address)
	if err != nil {
		return err
	}
	grpcAuthn := grpclib.NewGrpcAuthenticator(m.authn)
	m.grpcService = grpclib.NewGrpcServiceServer(
		lis, m.tlsCert, m.serviceName, grpcAuthn, time.Minute)

	m.grpcService.CreateStream(grpcapi.StreamNameNotification, m.ServeStreamConnection)
	// m.grpcService.AddStream(grpcapi.StreamNameRequestResponse, m.ServeStreamConnection)

	m.Init(DefaultUDSModuleID,
		transports.ProtocolTypeHiveotGrpc,
		transports.SubprotocolHiveotGrpc,
		m.connectURL, m.authn)

	err = m.grpcService.Start()
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
	m.grpcService.Stop()
}

// GRPC server using UDS or TCP sockets.
//
// Server side listening uses net.Listen This accepts a scheme that is "unix" for UDS
// sockets or "tcp" for TCP sockets.
// The address part of the URL is the full path to the socket, eg /run/myapp.sock, or
// in case of TCP sockets, the host and port, eg localhost:50051 or simply :50051.
//
// connectURL is the URL to listen on, e.g. scheme://address used in creating a net.listener
// tlsCert is the TLS certificate to use for secure connections, or nil for insecure
// authn is the authenticator for verifying the client token
// respTimeout is the time the server waits for a response when sending requests. defaults to 3sec
func NewHiveotGrpcTransportServer(
	connectURL string, tlsCert *tls.Certificate,
	authn transports.IAuthenticator, respTimeout time.Duration) *GrpcTransportServer {

	srv := &GrpcTransportServer{
		authn:       authn,
		connectURL:  connectURL,
		tlsCert:     tlsCert,
		respTimeout: respTimeout,
		serviceName: grpcapi.GrpcTransportServiceName,
	}
	return srv
}

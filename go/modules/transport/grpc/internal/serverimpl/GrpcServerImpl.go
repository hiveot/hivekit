package serverimpl

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/modules/transport"
	grpctransport "github.com/hiveot/hivekit/go/modules/transport/grpc"
	grpclib "github.com/hiveot/hivekit/go/modules/transport/grpc/internal"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/teris-io/shortid"
	"google.golang.org/grpc"
)

const DefaultUDSModuleID = "hiveot-uds"

// GrpcServerImpl is the transport server using gRPC connections.
//
// This implements both ITransportServer and IHiveModule interfaces.
// The embedded TransportServerBase is used for managing connections and forwarding messages to sinks.
type GrpcServerImpl struct {
	*transport.TransportServerBase
	// Authenticate
	authenticator api.IAuthenticator

	tlsCert *tls.Certificate

	caCert *x509.Certificate

	grpcService *grpclib.GrpcServiceServer

	respTimeout time.Duration

	// the service name the streams are published under
	serviceName string
}

// The grpc service callback handler for incoming stream connections.
// This creates a new transport connection for the stream and blocks until the stream is closed.
func (m *GrpcServerImpl) ServeStreamConnection(
	clientID string, cid string, grpcStream grpc.ServerStream) error {

	// authentication???

	// Create a hiveot transport connection for this stream.
	c := StartGrpcTransportConnection(
		clientID, cid, grpcStream, m.ForwardRequest, m.ForwardNotification)
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
func (m *GrpcServerImpl) Start() (err error) {

	slog.Info("Start: Starting grpc transport server",
		slog.String("connectURL", m.GetConnectURL()))

	address := m.GetConnectURL()
	network := "tcp"
	// start listening on unix sockets. Make sure the directory exists and the socket file doesn't.
	if strings.HasPrefix(address, "unix") {
		// m.connectURL is the same for the client
		network = "unix"
		address = strings.TrimPrefix(address, "unix://")
		socketDir := filepath.Dir(address)
		_ = os.Remove(address)
		err = os.MkdirAll(socketDir, 0700)
		if err != nil {
			return err
		}

	} else {
		// address is a tcp network tcp://ip:port
		// gRPC clients do not support tcp scheme. remove it and use the server IP
		address = strings.TrimPrefix(address, "tcp://")
		network = "tcp"
	}

	lis, err := net.Listen(network, address)
	if err != nil {
		return err
	}
	grpcAuthn := grpclib.NewGrpcAuthenticator(m.authenticator)
	m.grpcService = grpclib.NewGrpcServiceServer(
		lis, m.tlsCert, m.caCert, m.serviceName, grpcAuthn, time.Minute)

	m.grpcService.CreateStream(grpctransport.StreamNameNotification, m.ServeStreamConnection)
	// m.grpcService.AddStream(grpcapi.StreamNameRequestResponse, m.ServeStreamConnection)

	err = m.grpcService.Start()
	if err != nil {
		lis.Close()
		return err
	}

	return err
}

// Stop any running actions
func (m *GrpcServerImpl) Stop() {
	slog.Info("Stop: Stopping grpc transport server")
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
//	address is the URL to listen on, e.g. scheme://address used in creating a net.listener
//	 use "" for default unix socket path
//	tlsCert is the server TLS certificate to use for secure connections, or nil for insecure
//	caCert *x509.Certificate is the CA certificate to validate client auth. nil to ignore
//	authn is the authenticator for verifying the client token
//	respTimeout is the time the server waits for a response when sending requests. defaults to 3sec
func NewGrpcServerImpl(
	address string, tlsCert *tls.Certificate, caCert *x509.Certificate,
	authn api.IAuthenticator, respTimeout time.Duration) *GrpcServerImpl {

	// cleanup the connect URL into one of these:
	// UDS: unix://path/to/sock
	// TCP: tcp://address:port

	// connectURL is the client endpoint to connect to
	connectURL := address

	if address == "" {
		connectURL = grpctransport.DefaultGrpcURL
	} else if strings.HasPrefix(address, "unix") {
		// no change
	} else {
		// the dns scheme allows including of a DNS server. This is not supported.
		if strings.HasPrefix(address, "dns") {
			// dns scheme use triple slashes
			address = strings.TrimPrefix(address, "dns:///")

		} else if strings.HasPrefix(address, "tcp") {
			// gRPC *clients* do not support tcp scheme. remove it and use the server IP
			address = strings.TrimPrefix(address, "tcp://")
		} else {
			// some unknown or missing scheme.
			// remove the prefix if any and just use the address with tcp
			parts := strings.Split(address, "://")
			address = parts[len(parts)-1]
		}
		// if address is just a port then include the outbound IP for connecting to
		if strings.HasPrefix(address, ":") {
			// :port -> tcp://outboundIP:port
			connectURL = fmt.Sprintf("tcp://%s%s", utils.GetOutboundIP("").String(), address)
		} else {
			// full address to connect;
			// tcp:   tcp://host:port
			// unix:  unix://path/to/sock
			connectURL = fmt.Sprintf("tcp://%s", address)
		}
	}
	thingID := grpctransport.HiveotGrpcServerModuleType + "-" + shortid.MustGenerate()
	srv := &GrpcServerImpl{
		TransportServerBase: transport.NewTransportServerBase(thingID, connectURL, authn),
		authenticator:       authn,
		caCert:              caCert,
		tlsCert:             tlsCert,
		respTimeout:         respTimeout,
		serviceName:         grpctransport.GrpcTransportServiceName,
	}
	return srv
}

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
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/api/vocab"
	"github.com/hiveot/hivekit/go/cells/transport"
	grpctransport "github.com/hiveot/hivekit/go/cells/transport/grpc"
	grpclib "github.com/hiveot/hivekit/go/cells/transport/grpc/internal"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/teris-io/shortid"
	"google.golang.org/grpc"
)

// GrpcServerImpl is the transport server using gRPC connections.
//
// This implements both ITransportServer and IHiveCell interfaces.
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

	// the TD describing this server
	serverTD *td.TD

	// the gRPC subprotocol, eg unix or tcp
	subprotocol string
}

// GetTD returns the server TD, containing connection and authentication information
func (srv *GrpcServerImpl) GetTD() *td.TD {
	return srv.serverTD
}

// The grpc service callback handler for incoming stream connections.
// This creates a new transport connection for the stream and blocks until the stream is closed.
func (srv *GrpcServerImpl) ServeStreamConnection(
	clientID string, cid string, grpcStream grpc.ServerStream) error {

	// authentication???

	// Create a hiveot transport connection for this stream.
	c := NewGrpcServerConnection(
		clientID, cid, grpcStream, srv.ForwardRequest, srv.ForwardNotification)
	c.SetTimeout(srv.respTimeout)

	srv.AddConnection(c)
	// must block until connection closes
	c.WaitUntilDisconnect()
	srv.RemoveConnection(c)
	return nil
}

// Start the server with the given configuration.
// The server will listen on the configured URL and handle incoming connections.
// This adapts the URL scheme "unix", "uds", or "tcp" to the appropriate network type for net.Listen
// and update the connectURL to match the scheme used for listening.
func (srv *GrpcServerImpl) Start() (err error) {

	slog.Info("Start: Starting grpc transport server",
		slog.String("connectURL", srv.GetConnectURL()))

	address := srv.GetConnectURL()
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
	grpcAuthn := grpclib.NewGrpcAuthenticator(srv.authenticator)
	srv.grpcService = grpclib.NewGrpcServiceServer(
		lis, srv.tlsCert, srv.caCert, srv.serviceName, grpcAuthn, time.Minute)

	srv.grpcService.CreateStream(grpctransport.StreamNameNotification, srv.ServeStreamConnection)
	// m.grpcService.AddStream(grpcapi.StreamNameRequestResponse, m.ServeStreamConnection)

	err = srv.grpcService.Start()
	if err != nil {
		lis.Close()
		return err
	}

	// create a TD describing this server along with its connection URL
	thingID := srv.GetThingID()
	srv.serverTD = td.NewTD(thingID, "gRPC server", vocab.DeviceTypeService)
	srv.AddTDSecForms(srv.serverTD, false)
	return err
}

// Stop any running actions
func (srv *GrpcServerImpl) Stop() {
	slog.Info("Stop: Stopping grpc transport server")
	srv.CloseAll()
	srv.grpcService.Stop()
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
	subProtocol := api.HiveotGrpcTcpSubprotocol

	if address == "" {
		connectURL = grpctransport.DefaultGrpcUnixURL
		subProtocol = api.HiveotGrpcUnixSubprotocol
	} else if strings.HasPrefix(address, "unix") {
		subProtocol = api.HiveotGrpcUnixSubprotocol
		// no change
	} else {
		// the dns scheme allows including of a DNS server. This is not supported.
		if strings.HasPrefix(address, "dns") {
			subProtocol = api.HiveotGrpcTcpSubprotocol
			// dns scheme use triple slashes
			address = strings.TrimPrefix(address, "dns:///")

		} else if strings.HasPrefix(address, "tcp") {
			// gRPC *clients* do not support tcp scheme. remove it and use the server IP
			address = strings.TrimPrefix(address, "tcp://")
			subProtocol = api.HiveotGrpcTcpSubprotocol
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
			subProtocol = api.HiveotGrpcTcpSubprotocol
		} else {
			// full address to connect;
			// tcp:   tcp://host:port
			// unix:  unix://path/to/sock
			connectURL = fmt.Sprintf("tcp://%s", address)
			subProtocol = api.HiveotGrpcTcpSubprotocol
		}
	}
	thingID := grpctransport.HiveotGrpcServerCellType + "-" + shortid.MustGenerate()
	srv := &GrpcServerImpl{
		TransportServerBase: transport.NewTransportServerBase(thingID, connectURL, authn),
		authenticator:       authn,
		caCert:              caCert,
		tlsCert:             tlsCert,
		respTimeout:         respTimeout,
		serviceName:         grpctransport.GrpcTransportServiceName,
		subprotocol:         subProtocol,
	}
	return srv
}

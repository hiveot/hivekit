package internal

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/vocab"
	"github.com/hiveot/hivekit/go/api/wot"
	"github.com/hiveot/hivekit/go/cells/transport"
	grpctransport "github.com/hiveot/hivekit/go/cells/transport/grpc"
	"github.com/hiveot/hivekit/go/cells/transport/grpc/grpclib"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/teris-io/shortid"
	"google.golang.org/grpc"
)

// GrpcServerImpl is the transport server using gRPC connections.
//
// This implements both ITransportServer and IHiveCell interfaces.
// The embedded TransportServerBase is used for managing connections and emitting messages to sinks.
type GrpcServerImpl struct {
	*transport.TransportServerBase

	// flag, include affordances when adding forms
	includeAffordances bool

	// Authenticate
	authenticator api.IAuthenticator

	tlsCert *tls.Certificate

	caCert *x509.Certificate

	grpcService *grpclib.GrpcServiceServer

	// the service name the streams are published under
	serviceName string

	// the TD describing this server
	serverTD *wot.TD

	// the gRPC subprotocol, eg unix or tcp
	subprotocol string
}

// GetTD returns the server TD, containing connection and authentication information
func (srv *GrpcServerImpl) GetTD() *wot.TD {
	return srv.serverTD
}

// The grpc service callback handler for incoming stream connections.
// This creates a new transport connection for the stream and blocks until the stream is closed.
func (srv *GrpcServerImpl) ServeStreamConnection(
	clientID string, cid string, grpcStream grpc.ServerStream) error {

	// authentication???

	// Create a hiveot transport connection for this stream.
	c := StartGrpcServerConnection(
		clientID, cid, grpcStream, srv.EmitRequest, srv.EmitNotification)

	srv.AddConnection(c)
	// must block until connection closes
	c.WaitUntilDisconnect()
	srv.RemoveConnection(c)
	return nil
}

// start the server with the given configuration.
// The server will listen on the configured URL and handle incoming connections.
// This adapts the URL scheme "unix", "uds", or "tcp" to the appropriate network type for net.Listen
// and update the connectURL to match the scheme used for listening.

func (srv *GrpcServerImpl) startServing(network string, address string) (err error) {

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
	thingID := srv.GetID()
	srv.serverTD = wot.NewTD(thingID, "gRPC server", vocab.DeviceTypeService)
	srv.AddTDSecForms(srv.serverTD)
	return err
}

// Stop any running actions
func (srv *GrpcServerImpl) Stop() {
	slog.Info("Stop: Stopping grpc transport server")
	srv.CloseAll()
	srv.grpcService.Stop()
}

// Return a ready-to-use GRPC server using UDS or TCP sockets.
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
func StartGrpcServerImpl(
	address string, tlsCert *tls.Certificate, caCert *x509.Certificate,
	authn api.IAuthenticator) (*GrpcServerImpl, error) {

	slog.Info("startServing: Starting grpc transport server",
		slog.String("address", address))

	// cleanup the connect URL into one of these:
	// UDS: unix:///path/to/sock
	// TCP: tcp://address:port or dns:///address:port

	// connectURL is the client endpoint to connect to
	// connectURL := address
	subProtocol := api.HiveotGrpcTcpSubprotocol

	parts, err := url.Parse(address)
	if err != nil {
		return nil, err
	}
	network := "tcp"       // tcp or unix
	connAddr := parts.Host // host:port or /path
	subProtocol = api.HiveotGrpcTcpSubprotocol
	connectURL := address

	switch parts.Scheme {
	case "unix":
		subProtocol = api.HiveotGrpcUnixSubprotocol
		network = "unix"
		// start listening on unix sockets. Make sure the directory exists and the socket file doesn't.
		connAddr = parts.Path
		socketDir := filepath.Dir(connAddr)
		_ = os.Remove(connAddr)
		err = os.MkdirAll(socketDir, 0700)
		if err != nil {
			return nil, err
		}
		// no change
	case "dns":
		subProtocol = api.HiveotGrpcTcpSubprotocol
		// DNS prefix uses extra 'domain' slash, remove it
		// url.Parse thinks the remainder is a path, so manually remove the slash
		connAddr = strings.TrimPrefix(parts.Path, "/")
		network = "tcp"
		connectURL = address
	default:
		subProtocol = api.HiveotGrpcTcpSubprotocol
		network = "tcp"
		if parts.Host == "" {
			connectURL = fmt.Sprintf("dns:///%s%s", utils.GetOutboundIP("").String(), address)
			connAddr = ":"
		} else {
			connAddr = parts.Host
			connectURL = address
		}
	}
	thingID := grpctransport.HiveotGrpcServerCellType + "-" + shortid.MustGenerate()
	srv := &GrpcServerImpl{
		TransportServerBase: transport.NewTransportServerBase(thingID, connectURL, authn),
		authenticator:       authn,
		caCert:              caCert,
		tlsCert:             tlsCert,
		serviceName:         grpctransport.GrpcTransportServiceName,
		subprotocol:         subProtocol,
	}

	err = srv.startServing(network, connAddr)
	return srv, err
}

package grpctransportpkg

import (
	"crypto/tls"
	"time"

	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/factory"
	"github.com/hiveot/hivekit/go/modules/transports"
	grpctransport "github.com/hiveot/hivekit/go/modules/transports/grpc"
	"github.com/hiveot/hivekit/go/modules/transports/grpc/internal/grpcserver"
)

// NewHiveotGrpcServer creates a hiveot gRPC transport server.
//
// # This uses the HiveOT RRN messages as the payload.
//
// connectURL is the URL to listen on, e.g. unix:///tmp/hivekit.sock or tcp://localhost:50051
// tlsCert is the TLS certificate to use for secure connections, or nil for insecure
// authn is the authenticator to validate incoming connections
// respTimeout is the time the server waits for a response when receiving requests. defaults to 3sec
//
// Use SetRequestSink to set the handler for requests send by consumers
// Use SetNotificationSink to set the handler for notifications send by agents.
func NewHiveotGrpcServer(
	connectURL string, tlsCert *tls.Certificate, authn transports.IAuthenticator, respTimeout time.Duration) grpctransport.IGrpcTransportServer {
	return grpcserver.NewHiveotGrpcTransportServer(connectURL, tlsCert, authn, respTimeout)
}

// Create a new instance of the hiveot gRPC server using the factory environment
func NewHiveotGrpcServerFactory(f factory.IModuleFactory) modules.IHiveModule {
	// TODO: determine a good default
	connectURL := "unix:///var/hiveot/hivekit.sock"
	env := f.GetEnvironment()
	tlsCert, err := env.GetServerCert()
	_ = err
	authenticator := f.GetAuthenticator()
	m := NewHiveotGrpcServer(connectURL, tlsCert, authenticator, env.RpcTimeout)
	return m
}

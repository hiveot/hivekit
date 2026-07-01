package grpcpkg

import (
	"crypto/tls"
	"crypto/x509"
	"time"

	"github.com/hiveot/hivekit/go/api"
	grpctransport "github.com/hiveot/hivekit/go/modules/transport/grpc"
	internalserver "github.com/hiveot/hivekit/go/modules/transport/grpc/internal/server"
)

// NewHiveotGrpcServer creates a hiveot gRPC transport server.
//
// This uses the HiveOT RRN messages as the payload.
//
//	connectURL is the URL to listen on, e.g. "" for default
//	 for example  unix:///tmp/hivekit.sock or tcp://localhost:50051
//	tlsCert is the TLS certificate to use for secure connections, or nil for insecure
//	caCert for validating client certificate auth, nil to not support client certs
//	authn is the authenticator to validate incoming connections
//	respTimeout is the time the server waits for a response when receiving requests. defaults to 3sec
//
// Use SetRequestSink to set the handler for requests send by consumers
// Use SetNotificationSink to set the handler for notifications send by exposed things.
func NewHiveotGrpcServer(
	connectURL string, tlsCert *tls.Certificate, caCert *x509.Certificate,
	authn api.IAuthenticator, respTimeout time.Duration) grpctransport.IGrpcTransportServer {

	return internalserver.NewGrpcServer(connectURL, tlsCert, caCert, authn, respTimeout)
}

// Create a new instance of the hiveot gRPC server using the factory environment
func NewHiveotGrpcServerFactory(f api.IModuleFactory, md *api.ModuleDefinition) (api.IHiveModule, error) {
	// TODO: determine a good default
	env := f.GetEnvironment()
	serverCert, err := env.GetTLSCert()
	caCert, err := env.GetCACert()
	_ = err

	grpcURL := grpctransport.DefaultGrpcURL
	if md.Config != nil {
		grpcConfig, ok := md.Config.(grpctransport.GrpcConfig)
		if ok {
			grpcURL = grpcConfig.URL
		}
	}

	m := NewHiveotGrpcServer(grpcURL, serverCert, caCert, f.GetAuthenticator(), env.RpcTimeout)
	return m, nil
}

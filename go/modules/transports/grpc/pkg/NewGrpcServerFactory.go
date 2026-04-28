package grpcpkg

import (
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/factory"
)

// Create a new instance of the hiveot gRPC server using the factory environment
func NewGrpcServerFactory(f factory.IModuleFactory) modules.IHiveModule {
	// TODO: determine a good default
	connectURL := "unix:///var/hiveot/hivekit.sock"
	env := f.GetEnvironment()
	tlsCert, err := env.GetServerCert()
	_ = err
	authenticator := f.GetAuthenticator()
	m := NewHiveotGrpcServer(connectURL, tlsCert, authenticator, env.RpcTimeout)
	return m
}

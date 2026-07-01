package discoverypkg

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/modules/transport/discovery"
	internal "github.com/hiveot/hivekit/go/modules/transport/discovery/internal/client"
)

// NewDiscoveryClient creates a new instance of a discovery client
//
// appEnv is optional. On Start it will be updated with the discovered directory and server.
// discoOnStart runs a directory discovery on startup.
func NewDiscoveryClient(appEnv *api.AppEnvironment, discoOnStart bool) discovery.IDiscoveryClient {
	cl := internal.NewDiscoveryClientImpl(appEnv, discoOnStart)
	return cl
}

// NewDiscoveryClientFactory creates a new instance of a discovery client for
// use by the factory.
//
// This automatically runs discovery of things on the network on Start()
//
// Intended to be used by a client side factory recipe to automatically discover devices.
func NewDiscoveryClientFactory(f api.IModuleFactory, md *api.ModuleDefinition) (api.IHiveModule, error) {
	appEnv := f.GetEnvironment()
	cl := NewDiscoveryClient(appEnv, true)
	// nothing else to do here right now

	return cl, nil
}

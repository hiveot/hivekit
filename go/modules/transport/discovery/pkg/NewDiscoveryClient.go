package discoverypkg

import (
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/factory"
	"github.com/hiveot/hivekit/go/modules/transport/discovery"
	internal "github.com/hiveot/hivekit/go/modules/transport/discovery/internal/client"
)

// NewDiscoveryClient creates a new instance of a discovery client
//
// appEnv is optional. On Start it will be updated with the discovered directory and server.
func NewDiscoveryClient(appEnv *factory.AppEnvironment) discovery.IDiscoveryClient {
	cl := internal.NewDiscoveryClient(appEnv)
	return cl
}

// NewDiscoveryClientFactory creates a new instance of a discovery client for
// use by the factory.
// On start this updates the factory environment with the directory server URL.
//
// Intended to be used by a client side factory recipe to automatically discover the
// directory TDD and gateway TD.
func NewDiscoveryClientFactory(f factory.IModuleFactory) (modules.IHiveModule, error) {
	appEnv := f.GetEnvironment()
	cl := NewDiscoveryClient(appEnv)
	// nothing else to do here right now

	return cl, nil
}

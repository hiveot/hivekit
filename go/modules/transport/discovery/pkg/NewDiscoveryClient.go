package discoverypkg

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/modules/transport/discovery"
	internal "github.com/hiveot/hivekit/go/modules/transport/discovery/internal/client"
)

// NewDiscoveryClient creates a new instance of a discovery client
//
// If an appEnv is provided and its DirectoryURL is empty, and discoOnStart is enabled
// then Start will run in initial directory discovery and update appEnv with the
// resulting directory.
//
// This provides automatic discovery of a directory for a consumer that uses this module,
// while still be able to provide a commandline override of the directory URL.
func NewDiscoveryClient(appEnv *api.AppEnvironment, discoOnStart bool) discovery.IDiscoveryClient {
	cl := internal.NewDiscoveryClientImpl(appEnv, discoOnStart)
	return cl
}

// NewDiscoveryClientFactory creates a new instance of a discovery client for
// use by the factory.
//
// Intended to be used by a client side factory recipe to automatically discover devices.
func NewDiscoveryClientFactory(f api.IModuleFactory, md *api.ModuleDefinition) (api.IHiveModule, error) {
	appEnv := f.GetEnvironment()
	cl := NewDiscoveryClient(appEnv, false)
	// nothing else to do here right now

	return cl, nil
}

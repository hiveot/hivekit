package discovery_client

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/cells/transport/discovery"
	"github.com/hiveot/hivekit/go/cells/transport/discovery/internal/clientimpl"
)

// StartDiscoveryClient creates a new instance of a discovery client
//
// If an appEnv is provided and its DirectoryURL is empty, and discoOnStart is enabled
// then Start will run in initial directory discovery and update appEnv with the
// resulting directory.
//
// This provides automatic discovery of a directory for a consumer that uses this client,
// while still be able to provide a commandline override of the directory URL.
func StartDiscoveryClient(
	appEnv *api.HiveEnvironment, discoOnStart bool) (discovery.IDiscoveryClient, error) {

	return clientimpl.StartDiscoveryClientImpl(appEnv, discoOnStart)
}

// StartDiscoveryClientFactory creates a new instance of a discovery client for
// use by the factory.
//
// Intended to be used by a client side factory recipe to automatically discover devices.
func StartDiscoveryClientFactory(f api.ICellFactory, md *api.CellDefinition) (api.IHiveCell, error) {
	appEnv := f.GetEnvironment()
	return StartDiscoveryClient(appEnv, false)
}

package discovery_client

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/cells/transport/discovery"
	"github.com/hiveot/hivekit/go/cells/transport/discovery/internal/clientimpl"
)

// NewDiscoveryClient returns a ready-to-use instance of a discovery client
//
// Call DiscoverThings or DiscoverDirectories to start the discovery process.
//
// If an appEnv is provided and its DirectoryURL is empty, and discoOnStart is enabled
// then Start will run in initial directory discovery and update appEnv with the
// resulting directory.
//
// This provides automatic discovery of a directory for a consumer that uses this client,
// while still be able to provide a commandline override of the directory URL.
func NewDiscoveryClient(
	appEnv *api.HiveEnvironment, discoOnStart bool) (discovery.IDiscoveryClient, error) {

	return clientimpl.NewDiscoveryClientImpl(appEnv, discoOnStart)
}

// NewDiscoveryClientFactory returns a ready-to-use instance of a discovery client
// for use by the factory.
//
// Intended to be used by a client side factory recipe to automatically discover devices.
func NewDiscoveryClientFactory(f api.ICellFactory, md *api.CellDefinition) (api.IHiveCell, error) {
	appEnv := f.GetEnvironment()
	return NewDiscoveryClient(appEnv, false)
}

package discovery_client

import (
	"time"

	"github.com/grandcat/zeroconf"
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/cells/transport/discovery"
	"github.com/hiveot/hivekit/go/cells/transport/discovery/client/internal"
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
	env *api.HiveEnvironment, discoOnStart bool) (discovery.IDiscoveryClient, error) {

	return internal.NewDiscoveryClientImpl(env, discoOnStart)
}

// NewDiscoveryClientFactory returns a ready-to-use instance of a discovery client
// for use by the factory.
//
// Intended to be used by a client side factory recipe to automatically discover devices.
func NewDiscoveryClientFactory(f api.ICellFactory, md *api.CellDefinition) (api.IHiveCell, error) {
	env := f.GetEnvironment()
	return NewDiscoveryClient(env, true)
}

// DnsSDScan scans zeroconf publications on local domain
//
// The zeroconf library does not support browsing of all services, but a workaround is
// to search the service types with "_services._dns-sd._udp" then query each of the service types.
//
// The provided callback is concurrent safe. The scan does not return while the callback is invoked.
//
// results are handled through a callback until the waitTime ends or the callback returns stop=true
//
//	instanceName to look for, or "" for all possible instances
//	serviceType to look for in format "_{serviceName}._tcp", or "" to discover all service types (not all services)
//	waitTime with duration to wait while collecting results. Default is 3 seconds
//	cb is the optional callback invoked when a result is found
func DnsSDScan(instanceName string, serviceType string, waitTime time.Duration,
	cb func(*zeroconf.ServiceEntry) (stop bool)) (records []*zeroconf.ServiceEntry, err error) {
	return internal.DnsSDScan(instanceName, serviceType, waitTime, cb)
}

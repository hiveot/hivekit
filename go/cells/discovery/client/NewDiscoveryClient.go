package discovery_client

import (
	"time"

	"github.com/grandcat/zeroconf"
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/cells/discovery"
	"github.com/hiveot/hivekit/go/cells/discovery/client/internal"
)

// NewDiscoveryClient returns a ready-to-use instance of a discovery client.
// The primary use is to discovery a directory, either automatically or manually.
//
// If env.DirTDURL is set then the TD it points to is downloaded.
// If no directory is downloaded and discoOnStart is set then discovery will
// be started and env is updated with the first discovered directory.
//
// Call DiscoverThings or DiscoverDirectories to manually run the discovery
// process.
//
// This provides automatic discovery of a directory for a consumer that uses this client,
// while still be able to provide a commandline override of the directory URL.
func NewDiscoveryClient(
	env *api.AppEnvironment, discoOnStart bool) (discovery.IDiscoveryClient, error) {

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

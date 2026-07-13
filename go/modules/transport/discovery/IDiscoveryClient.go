package discovery

import (
	"fmt"
	"time"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/td"
)

const (
	// The discovery client module type for including in a module chain
	DiscoveryClientModuleType = "discovery-client"

	// Action request to discover a directory TDD.
	// Output: JSON with directory TD.
	//
	// This action is intended for applications to request 'rediscovery' of TD Directories
	// and Things after the chain has started.
	DiscoverDirectoryAction = "discoverDirectory"
)

type DiscoveryResult struct {
	Addr        string // IP or hostname of the server
	Port        int    // port the server listens on
	IsDirectory bool   // URL is that of a Thing Directory
	IsThing     bool   // URL is of a Thing
	Instance    string
	// predefined WoT discovery parameters
	Schema string // Schema part of the URL
	Type   string // Thing or Directory
	TD     string // absolute pathname of the TD or TDD
	// hiveot connection endpoints
	AuthEndpoint string            // authentication service endpoint
	SSEEndpoint  string            // Http/SSE-SC transport protocol
	WSSEndpoint  string            // Websocket transport
	Params       map[string]string // optional parameters
}

// Return the URL contained in the discovery record.
// This usually points to the Thing TD record. See also DownloadTD(url)
func (dr *DiscoveryResult) AsURL() string {
	fullUrl := fmt.Sprintf("%s://%s:%d%s", dr.Schema, dr.Addr, dr.Port, dr.TD)
	return fullUrl
}

// IDiscoveryClient is the interface of discovery client.
// This module is for discovering Thing TD's or Directory TDD's on the local network.
type IDiscoveryClient interface {
	api.IHiveModule

	// DiscoverDirectories supports introduction mechanisms to bootstrap the WoT discovery
	// process and returns a list of discovered directory TD URLs with the wot service name.
	//
	// Intended for clients that need to find one or more WoT directories.
	//
	//	searchTime is the time to search for.
	//	cb is the optional callback to call for each discovered thing. It should
	//  return true to stop or false to continue searching up until the searchTime.
	//
	// This returns a list of all discoveries or an error if discovery was unable to run
	DiscoverDirectories(searchTime time.Duration, cb func(*DiscoveryResult) bool) ([]*DiscoveryResult, error)

	// Discover directories and load their TD's
	// If a TD cannot be downloaded it is ignored.
	DiscoverDirectoryTDs(searchTime time.Duration) ([]*DiscoveryResult, []*td.TD)

	// DiscoverDirectory returns the discovery record of the first discovered directory
	//
	//	instanceName is the optional name of a non-default service instance.
	//   this defaults to WOT_DIRECTORY_SERVICE_TYPE (_directory._sub._wot._tcp)
	//	maxWaitTime defaults to 3 seconds
	//
	//	This returns the record or nil if none was found within the search time.
	//	This returns an error if it wasn't possible to run discovery.
	DiscoverFirstDirectory(
		instanceName string, maxWaitTime time.Duration) (rec0 *DiscoveryResult, err error)

	// DiscoverDirectoryTD returns the TD of the first discovered directory
	// This uses the service type WOT_DIRECTORY_SERVICE_TYPE (_directory._sub._wot._tcp)
	//
	//	instanceName is an optional filter name of a specific directory instance, or "" for default.
	//	maxWaitTime defaults to 3 seconds
	//
	//	This returns the TD, its JSON, if found
	//	This returns an error if it wasn't possible to run discovery.
	DiscoverFirstDirectoryTD(
		instanceName string, maxWaitTime time.Duration) (tdoc *td.TD, tddJson string, err error)

	// DiscoverThings returns a list of all discovery records of all WoT compatible devices,
	// including Things, Directories and Gateways.
	//
	// This uses the service type WOT_DEVICE_SERVICE_TYPE (_wot._tcp)
	//
	//	instanceName is an optional filter name of a specific thing instance, or "" for default.
	//	searchTime defaults to 3 seconds
	//	cb is the optional callback to call for each discovered thing. It should
	//  return true to stop or false to continue searching up until the searchTime.
	//
	//	This returns a list of the records
	//	This returns an error if it wasn't possible to run discovery.
	DiscoverThings(instanceName string, searchTime time.Duration,
		cb func(*DiscoveryResult) bool) (recs []*DiscoveryResult, err error)

	// Discover Things and download their TD
	DiscoverThingTDs(instanceName string, searchTime time.Duration,
		cb func(*td.TD) bool) ([]*DiscoveryResult, []*td.TD)

	// DownloadTD a TD document from a discovery record.
	// Intended to obtain the TD of a discovered directory or thing.
	//
	// tdURL points to the discovery spec http well-known endpoint address.
	//
	// This returns the TD, its JSON or an error if none is found
	LoadTD(tdURL string) (tdoc *td.TD, tdJSON string, err error)
}

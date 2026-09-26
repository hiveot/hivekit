package discovery

import (
	"fmt"
	"time"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/td"
)

const (
	// The discovery client cell type for including in a cell chain
	DiscoveryClientCellType = "discovery-client"

	// Action request to discover a directory TDD.
	// Output: JSON with directory TD.
	//
	// This action is intended for applications to request 'rediscovery' of TD Directories
	// and Things after the chain has started.
	DiscoverDirectoryAction = "discoverDirectory"
)

type DiscoveryResult struct {
	Addr        string // IP address of the server
	Hostname    string // hostname
	Port        int    // port the server listens on
	Service     string // DNS-SD service field
	IsDirectory bool   // URL is that of a Thing Directory
	IsThing     bool   // URL is of a Thing
	Instance    string
	// predefined WoT discovery parameters
	Schema string            // Schema part of the URL
	Type   string            // Thing, Directory or Gateway
	TD     string            // absolute pathname of the TD or TDD
	Params map[string]string // optional parameters

	// Experimental: additional hiveot connection endpoints
	// These are intended for discovering gateways
	AuthEndpoint string // authentication service endpoint
	SSEEndpoint  string // Http/SSE-SC transport protocol
	WSSEndpoint  string // Websocket transport
}

// Return the URL contained in the discovery record.
// This usually points to the Thing TD record. See also DownloadTD(url)
func (dr *DiscoveryResult) AsURL() string {
	fullUrl := fmt.Sprintf("%s://%s:%d%s", dr.Schema, dr.Addr, dr.Port, dr.TD)
	return fullUrl
}

// IDiscoveryClient is the interface of discovery client.
// This client is for discovering Thing TD's or Directory TDD's on the local network.
type IDiscoveryClient interface {
	api.IHiveCell

	// Return the first discovered TD that matches the searchID and type.
	//
	//	searchID optional filter on a specific instance name or directory thingID
	//	thingType optional filter DISCO_TYPE_THING|DIRECTORY|GATEWAY
	DiscoverFirstTD(
		searchID string, thingType string, maxWaitTime time.Duration) *td.TD

	// Return the discovery record of the first thing that matches the searchID and type.
	DiscoverFirstThing(
		instanceName string, thingType string, maxWaitTime time.Duration) *DiscoveryResult

	// Discover Things and download their TD.
	//
	// This separates directories from devices.
	// If a TD cannot be read this includes nil in the TD results so the
	// records table matches the TD table.
	//
	//	searchID optionally filters on a specific instance name or directory thingID
	//	thingType optional filter DISCO_TYPE_THING|DIRECTORY|GATEWAY
	//	first stops on first valid result
	//	maxWaitTime is maximum time to wait for search to complete
	//
	// This returns the matching directory, gateway and thing TDs
	DiscoverTDs(instanceName string, thingType string, first bool, searchTime time.Duration,
		cb func(*td.TD) bool) (
		dirs []*DiscoveryResult, dirTDs []*td.TD,
		things []*DiscoveryResult, thingTDs []*td.TD)

	// DiscoverThings returns a list of all discovery records of all WoT compatible devices,
	// including Things, Directories and Gateways.
	//
	// This uses the service type WOT_DEVICE_SERVICE_TYPE (_wot._tcp)
	//
	//	instanceName is an optional filter name of a specific thing instance, or "" for default.
	//	thingType, DISCO_TYPE_THING|DIRECTORY|GATEWAY
	//	searchTime defaults to 3 seconds
	//	cb is the optional callback to call for each discovered thing. It should
	//  return true to stop or false to continue searching up until the searchTime.
	//
	//	This returns a list of the records
	//	This returns an error if it wasn't possible to run discovery.
	DiscoverThings(instanceName string, thingType string, first bool, searchTime time.Duration,
		cb func(*DiscoveryResult) bool) (recs []*DiscoveryResult, err error)

	// DownloadTD a TD document from a discovery record.
	// Intended to obtain the TD of a discovered directory or thing.
	//
	// tdURL points to the download URL of the TD document
	//
	// This returns the TD, its JSON or an error if none is found
	LoadTD(tdURL string) (tdoc *td.TD, tdJSON string, err error)
}

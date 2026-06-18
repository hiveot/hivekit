package discovery

import (
	"fmt"
	"time"
)

// The discovery module types
const (
	DiscoveryClientModuleType = "discovery-client"
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
	// DiscoverDevices returns a list of discovery records of WoT compatible devices.
	//
	//	instanceName is the optional name of the directory instance, "" for default
	//   this defaults to WOT_DEVICE_SERVICE_TYPE (_wot._tcp)
	//	searchTime defaults to 3 seconds
	//
	//	This returns a list of the records
	//	This returns an error if it wasn't possible to run discovery.
	DiscoverThings(instanceName string, searchTime time.Duration) (recs []*DiscoveryResult, err error)

	// DiscoverDirectory returns the discovery record of the first discovered directory
	//
	//	instanceName is the optional name of a non-default service instance.
	//   this defaults to WOT_DIRECTORY_SERVICE_TYPE (_directory._sub._wot._tcp)
	//	searchTime defaults to 3 seconds
	//
	//	This returns the record or nil if none was found within 3 seconds.
	//	This returns an error if it wasn't possible to run discovery.
	DiscoverFirstDirectory(instanceName string, searchTime time.Duration) (rec0 *DiscoveryResult, err error)

	// DiscoverFirstGateway returns the discovery record if the first gateway server.
	//
	// To distinguish a gateway from other IoT devices it uses a predefined serviceID,
	// defined in discovery.DefaultGatewayServiceID.
	//
	// A custom instance name can be provided or "" for default.
	//
	//	instanceName is the optional name of the directory instance, "" for default
	//   this defaults to WOT_DIRECTORY_SERVICE_TYPE (_directory._sub._wot._tcp)
	//	searchTime defaults to 3 seconds
	//
	//	This returns the record or nil if none was found within 3 seconds.
	//	This returns an error if it wasn't possible to run discovery.
	DiscoverFirstGateway(instanceName string, searchTime time.Duration) (rec0 *DiscoveryResult, err error)
}

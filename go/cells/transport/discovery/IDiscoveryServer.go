package discovery

import "github.com/hiveot/hivekit/go/api"

// The discovery cell types
const (
	DiscoveryServerCellType = "discovery-server"
)

// DNS-SD service IDs
const (

	// Note the exploration mechanism has moved away from this service type. Instead,
	// DNS-SD is phase 1 of the introduction mechanism to obtain the TD url
	// to read over http. Use the TXT record TYPE field to distinguish directory from thing.

	// DNS-SD service sub-type for WoT Directory TD
	// See discovery specification: https://w3c.github.io/wot-discovery/#introduction-dns-sd-sec
	WOT_DIRECTORY_SUB_TYPE = "_directory._sub"

	// WoT doesnt define gateways in their discovery spec so use our own.
	// HIVEOT_GATEWAY_SUB_TYPE = "_gateway._sub"

	// DNS-SD general service type for all things
	WOT_SERVICE_TYPE = "_wot._tcp"
)

// additional fields in the discovery records
const AuthEndpoint = "login"
const WSSEndpoint = "wss"
const SSEEndpoint = "sse"

// WellKnownHttpPath contains the path to the digital twin directory
// TD document uses the 'well-known' path
const WellKnownHttpPath = "/.well-known/wot"

// Actions to serve discovery of a provided TD.
// Note: The discovery service triggers on requests with these actions regardless the thingID used.
//
// For example a cell chain with a service or device can publish its TD by sending a
// request message containing the TD downstream the chain.
const (
	// Action to serve a directory TD
	// Input: TDD Json document
	ServeDirectoryTDAction = "serveDirectoryTD"

	// Action to serve a Thing TD
	// Input: TD Json document
	ServeThingTDAction = "serveThingTD"
)

// IDiscoveryServer is the interface of a discovery server.
// This is a server for publishing the presence of the Thing or a Thing Directory.
//
// If this is used in a cell chain with a directory service then it must be placed
// after the directory service. Requests for writing TDs to the directory are
// intercepted by both the directory and the discovery server. If only a discovery
// server is present then it can be used to publish a stand-alone thing TD.
//
// Services that want to update a directory with its TD should use the
// DiscoveryClient instead.
type IDiscoveryServer interface {
	api.IHiveCell

	// ServeDirectoryTD serves the given directory TD on http at the well-known endpoint, and
	// publishes this using DNS-SD discovery.
	// Indended for use by devices that run a directory server.
	//
	// The TDD DNSSD service record is:
	//   _directory._sub._wot._tcp TXT td=/.well-known/wot; type=Directory;scheme=http
	//
	//	instanceName is the name under which the TDD is discoverable. Use "" for the default.
	//	tddJSON is the directory TD to make available in JSON format
	//
	// This fails if the http server isn't provided.
	ServeDirectoryTD(instanceName string, tddJSON string) (err error)

	// ServeThingTD serves the given thing TD on http at the well-known endpoint, and publishes
	// this using DNS-SD discovery.
	// Indended for use by things that run servers.
	//
	// The default TD DNSSD service record is:
	//   _wot._tcp TXT td=/.well-known/wot; type=Thing;scheme=http
	//
	// When a instanceName is provided (required when serving multiple records),
	// the the td path becomes: td=/.well-known/wot/{instanceName}
	//
	// This server also intercepts a directory updateTD request and publishes the TD
	// using this ServeThingTD handler, acting as a single-TD directory.
	//
	//	instanceName is the name under which the TD is discoverable. Use "" for the ThingID.
	//	tdJSON is the Thing TD to make available in JSON format
	//
	// This fails if the http server isn't provided.
	ServeThingTD(instanceName string, tdJSON string) (err error)
}

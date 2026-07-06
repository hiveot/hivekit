package discovery

import "github.com/hiveot/hivekit/go/api"

// The discovery module types
const (
	// DiscoveryServerModuleType is also its thingID
	DiscoveryServerModuleType = "discovery"
)

// DNS-SD service IDs
const (
	// WOT_UDP_DNSSD_TYPE = "_wot._udp"

	// DNS-SD service types for WoT Thing TD
	WOT_THING_SERVICE_TYPE = "_wot._tcp"

	// DNS-SD service types for WoT Directory TD
	// See discovery specification: https://w3c.github.io/wot-discovery/#introduction-dns-sd-sec
	WOT_DIRECTORY_SERVICE_TYPE = "_directory._sub._wot._tcp"

	// WoT doesnt define gateways in their discovery spec so use our own.
	// HIVEOT_GATEWAY_SERVICE_TYPE = "_gateway._sub._wot._tcp"
)

// additional fields in the discovery records
const AuthEndpoint = "login"
const WSSEndpoint = "wss"
const SSEEndpoint = "sse"

// WellKnownHttpPath contains the path to the digital twin directory
// TD document uses the 'well-known' path
const WellKnownHttpPath = "/.well-known/wot"

// Actions to serve discovery of a TD provided by a different module.
// Note: The discovery service triggers on requests with these actions regardless the thingID used.
//
// For example a module chain with a service or device module can publish its TD by sending a
// request message containing the TD downstream the chain.
const (
	// Action to start serving a directory TD
	// Input: TDD Json document
	ServeDirectoryTDAction = "serveDirectoryTD"

	// Action to start serving a Thing TD
	// Input: TD Json document
	ServeThingTDAction = "serveThingTD"
)

// IDiscoveryServer is the interface of a discovery server.
// This is a module that for publishing the presence of the Thing or a Thing Directory.
//
// If this is used in a module chain then the action to write a TD:
//
//	eg action CreateThingAction("createThing") is used to publish the included TD
//
// through discovery instead of forwarding it to a directory service.
type IDiscoveryServer interface {
	api.IHiveModule

	// ServeDirectoryTD serves the given directory TD on http at the well-known endpoint, and
	// publishes this using DNS-SD discovery.
	//
	// The TDD DNSSD service record is:
	//   _directory._sub._wot._tcp TXT td=/.well-known/wot; type=Directory;scheme=http
	//
	// This fails if the http server isn't provided.
	ServeDirectoryTD(dirTDJSON string) (err error)

	// ServeGatewayTD serves the given TD on http at the gateway endpoint, and publishes
	// this using DNS-SD discovery.
	//
	// The TD DNSSD service record is:
	//   _gateway._wot._tcp TXT td=/.well-known/gateway; type=Thing;scheme=http
	//
	// This server also intercepts a directory updateTD request and publishes the TD
	// using this ServeThingTD handler, acting as a single-TD directory.
	//
	// Indended for use by a gateway or hub.
	// ServeGatewayTD(gatewayTDJSON string) (err error)

	// ServeThingTD serves the given thing TD on http at the well-known endpoint, and publishes
	// this using DNS-SD discovery.
	//
	// The TD DNSSD service record is:
	//   _wot._tcp TXT td=/.well-known/wot; type=Thing;scheme=http
	//
	// This server also intercepts a directory updateTD request and publishes the TD
	// using this ServeThingTD handler, acting as a single-TD directory.
	//
	// Indended for use by things that run servers.
	ServeThingTD(thingTDJSON string) (err error)
}

package discovery

import "github.com/hiveot/hivekit/go/api"

// The discovery module types
const (
	ThingDiscoveryServerModuleType = "thingDiscovery"
)

// DNS-SD service IDs
const (
	// WOT_UDP_DNSSD_TYPE = "_wot._udp"

	// DNS-SD service types for WoT Thing TD
	WOT_THING_SERVICE_TYPE = "_wot._tcp"
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
	// Action to start serving a Thing TD
	// Input: TD Json document
	ServeThingTDAction = "serveThingTD"
)

// IThingDiscoveryServer is the interface of a discovery server.
// This is a module that for publishing the presence of the Thing.
//
// If this is used in a module chain then the action to write a TD.
//
//	eg action CreateThingAction("createThing") is used to publish the included TD
//
// through discovery instead of forwarding it to a directory service.
type IThingDiscoveryServer interface {
	api.IHiveModule

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

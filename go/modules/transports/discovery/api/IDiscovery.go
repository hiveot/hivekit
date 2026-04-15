package discoveryapi

import "github.com/hiveot/hivekit/go/modules"

// The discovery server module type
const DiscoveryServerModuleType = "discovery"

// The discovery module instance ID
const DefaultDiscoveryModuleID = "hivekit-discovery"

//const WOT_UDP_DNSSD_TYPE = "_wot._udp"

// DNS-SD service types for WoT Thing TD
const WOT_THING_SERVICE_TYPE = "_wot._tcp"

// DNS-SD service types for WoT Directory TD
// See discovery specification: https://w3c.github.io/wot-discovery/#introduction-dns-sd-sec
const WOT_DIRECTORY_SERVICE_TYPE = "_directory._sub._wot._tcp"

// additional fields in the discovery records
const AuthEndpoint = "login"
const WSSEndpoint = "wss"
const SSEEndpoint = "sse"

// DefaultHttpGetDirectoryTDPath contains the path to the digital twin directory
// TD document uses the 'well-known' path
const DefaultHttpGetDirectoryTDPath = "/.well-known/wot"

// IDiscoveryServer is the interface of a discovery server.
// This is a module that can be managed and controlled through request and notification messages.
type IDiscoveryServer interface {
	modules.IHiveModule

	// ServeDirectoryTDD registers the given directory TD with the http server and publishes
	// its endpoint using DNS-SD discovery.
	//
	// This fails if the http server isn't provided.
	ServeDirectoryTDD(dirTDJSON string) (err error)

	// ServeThingTD registers the given thing TD with the http server and publishes its
	// endpoint using DNS-SD discovery.
	//
	// Indended for use by things that run servers. (not recommended or needed when using a gateway)
	ServeThingTD(thingTDJSON string) (err error)
}

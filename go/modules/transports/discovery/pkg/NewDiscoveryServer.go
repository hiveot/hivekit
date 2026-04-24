package discoverypkg

import (
	"strings"

	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/factory"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/modules/transports/discovery"
	"github.com/hiveot/hivekit/go/modules/transports/discovery/internal"
)

// NewDiscoveryServer creates a new discovery server module instance.
//
//		httpServer is the server that serves the TD on the well-known endpoint.
//		endpoints are optional additional URLS to include in the DNS-SD discovery record
//		 where key is the schema "http", "wss", "sse-sc" and value the URL.
//	 serviceID to publish as. This is the module thingID
func NewDiscoveryServer(
	httpServer transports.IHttpServer, endpoints map[string]string, serviceID string) discovery.IDiscoveryServer {

	srv := internal.NewDiscoveryServer(httpServer, endpoints, serviceID)
	return srv
}

// Create a new instance of the discovery service using the factory environment
// This loads the http server.
// This creates a list of endpoints for each loaded transport server
func NewDiscoveryServerFactory(f factory.IModuleFactory) modules.IHiveModule {
	httpServer := f.GetHttpServer(true)
	endpoints := make(map[string]string)
	tps := f.GetTransportServers()
	for _, tp := range tps {
		connectURL := tp.GetConnectURL()
		parts := strings.Split(connectURL, ":")
		scheme := parts[0]
		endpoints[scheme] = connectURL
	}
	return NewDiscoveryServer(httpServer, endpoints, discovery.DefaultDiscoveryThingID)
}

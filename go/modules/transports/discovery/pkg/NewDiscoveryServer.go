package discoverypkg

import (
	"strings"

	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/factory"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/modules/transports/discovery"
	internalserver "github.com/hiveot/hivekit/go/modules/transports/discovery/internal/server"
)

// NewDiscoveryServer creates a new discovery server module instance.
//
//	serviceID to publish as. This is the module thingID
//	httpServer is the server that serves the TD on the well-known endpoint.
//	endpoints are optional additional URLS to include in the DNS-SD discovery record
//		 where key is the schema "http", "wss", "sse-sc" and value the URL.
func NewDiscoveryServer(serviceID string,
	httpServer transports.IHttpServer, endpoints map[string]string) discovery.IDiscoveryServer {

	srv := internalserver.NewDiscoveryServer(serviceID, httpServer, endpoints)
	return srv
}

// Create a new instance of the discovery service using the factory environment
// This loads the http server.
// This creates a list of endpoints for each loaded transport server
func NewDiscoveryServerFactory(f factory.IModuleFactory) (modules.IHiveModule, error) {
	httpServer := f.GetHttpServer(true)
	endpoints := make(map[string]string)
	tps := f.GetTransportServers()
	instanceID := discovery.DefaultDiscoveryThingID
	for _, tp := range tps {
		connectURL := tp.GetConnectURL()
		parts := strings.Split(connectURL, ":")
		scheme := parts[0]
		endpoints[scheme] = connectURL
	}
	return NewDiscoveryServer(instanceID, httpServer, endpoints), nil
}

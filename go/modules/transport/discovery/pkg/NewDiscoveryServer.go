package discoverypkg

import (
	"strings"

	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/factory"
	"github.com/hiveot/hivekit/go/modules/transport"
	"github.com/hiveot/hivekit/go/modules/transport/discovery"
	internalserver "github.com/hiveot/hivekit/go/modules/transport/discovery/internal/server"
)

// NewDiscoveryServer creates a new discovery server module instance.
//
//	thingID of the discovery server module. This defaults to the module type.
//	httpServer is the server that serves the TD on the well-known endpoint.
//	endpoints are optional additional URLS to include in the DNS-SD discovery record
//		 where key is the schema "http", "wss", "sse-sc" and value the URL.
func NewDiscoveryServer(thingID string,
	httpServer transport.IHttpServer, endpoints map[string]string) discovery.IDiscoveryServer {

	if thingID == "" {
		thingID = discovery.DiscoveryServerModuleType
	}
	srv := internalserver.NewDiscoveryServer(thingID, httpServer, endpoints)
	return srv
}

// Create a new instance of the discovery service using the factory environment
// The module type is used as the thingID.
// This loads the http server.
// This creates a list of endpoints for each loaded transport server
func NewDiscoveryServerFactory(f factory.IModuleFactory) (modules.IHiveModule, error) {
	httpServer := f.GetHttpServer(true)
	endpoints := make(map[string]string)
	tps := f.GetTransportServers()
	thingID := discovery.DiscoveryServerModuleType
	for _, tp := range tps {
		connectURL := tp.GetConnectURL()
		parts := strings.Split(connectURL, ":")
		scheme := parts[0]
		endpoints[scheme] = connectURL
	}
	return NewDiscoveryServer(thingID, httpServer, endpoints), nil
}

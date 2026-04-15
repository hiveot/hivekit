package discovery

import (
	"strings"

	factoryapi "github.com/hiveot/hivekit/go/factory/api"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/transports"
	discoveryapi "github.com/hiveot/hivekit/go/modules/transports/discovery/api"
	"github.com/hiveot/hivekit/go/modules/transports/discovery/internal"
)

// NewDiscoveryServer creates a new discovery server module instance.
//
//	dirTDJSON is the http path to the directory TD to be included in the discovery record.
//	httpServer is the server that serves the TD on the well-known endpoint.
//	endpoints are optional additional URLS to include in the DNS-SD discovery record
//	where key is the schema "http", "wss", "sse-sc" and value the URL.
func NewDiscoveryServer(httpServer transports.IHttpServer, endpoints map[string]string) discoveryapi.IDiscoveryServer {
	srv := internal.NewDiscoveryServer(httpServer, endpoints)
	return srv
}

// Create a new instance of the discovery service using the factory environment
// This loads the http server.
// This creates a list of endpoints for each loaded transport server
func NewDiscoveryServerFactory(f factoryapi.IModuleFactory) modules.IHiveModule {
	httpServer := f.GetHttpServer()
	endpoints := make(map[string]string)
	tps := f.GetTransportServers()
	for _, tp := range tps {
		connectURL := tp.GetConnectURL()
		parts := strings.Split(connectURL, ":")
		scheme := parts[0]
		endpoints[scheme] = connectURL
	}
	return NewDiscoveryServer(httpServer, endpoints)
}

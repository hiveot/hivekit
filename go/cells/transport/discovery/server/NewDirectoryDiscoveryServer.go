package discovery_server

import (
	"strings"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/cells/transport/discovery"
	serverimpl "github.com/hiveot/hivekit/go/cells/transport/discovery/internal/serverimpl"
)

// NewDirectoryDiscoveryServer creates a new discovery server instance.
//
// The optional instanceID is used both as the ThingID and as the instanceID
// in the discovery record.
//
//	instanceID of the discovery server. This defaults to {cell type}-{shortID}.
//	httpServer is the server that serves the TD on the well-known endpoint.
//	endpoints are optional additional URLS to include in the DNS-SD discovery record
//		 where key is the schema "http", "wss", "sse-sc" and value the URL.
func NewDirectoryDiscoveryServer(instanceID string,
	httpServer api.IHttpServer, endpoints map[string]string) discovery.IDirectoryDiscoveryServer {

	srv := serverimpl.NewDirectoryDiscoveryServerImpl(instanceID, httpServer, endpoints)
	return srv
}

// Create a new instance of the directory discovery service using the factory environment
// The cell type is used as the thingID.
// This loads the http server.
// This creates a list of endpoints for each loaded transport server
func NewDirectoryDiscoveryServerFactory(f api.ICellFactory, md *api.CellDefinition) (api.IHiveCell, error) {
	httpServer := f.GetHttpServer(true)
	endpoints := make(map[string]string)
	tps := f.GetTransportServers()

	for _, tp := range tps {
		connectURL := tp.GetConnectURL()
		parts := strings.Split(connectURL, ":")
		scheme := parts[0]
		endpoints[scheme] = connectURL
	}
	srv := NewDirectoryDiscoveryServer("", httpServer, endpoints)
	return srv, nil
}

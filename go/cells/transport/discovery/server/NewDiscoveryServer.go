package discovery_server

import (
	"os"
	"strings"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/cells/directory"
	"github.com/hiveot/hivekit/go/cells/transport/discovery"
	"github.com/hiveot/hivekit/go/cells/transport/discovery/internal/serverimpl"
)

// NewDiscoveryServer creates a new discovery server instance.
//
// The optional instanceID is used both as the ThingID and as the instanceID
// in the discovery record.
//
//	serviceName is the DNS-SD
//	httpServer is the server that serves the TD on the well-known endpoint.
//	tddJSON is the optional directory TDD as JSON to serve.
//	endpoints are optional additional URLS to include in the DNS-SD discovery record
//		 where key is the schema "http", "wss", "sse-sc" and value the URL.
func NewDiscoveryServer(serviceName string,
	httpServer api.IHttpServer,
	tddJSON string,
	endpoints map[string]string) discovery.IDiscoveryServer {

	srv := serverimpl.NewDiscoveryServerImpl(serviceName, httpServer, tddJSON, endpoints)
	return srv
}

// Create a new instance of the discovery server using the factory environment.
//
// When used in a cell chain together with a directory, this service must be placed
// after the directory in the chain, so it can find the directory to get its TDD,
// and prevent it from intercepting a CreateThing request send by services.
//
// This loads the http server.
// This creates a list of endpoints for each loaded transport server
func NewDiscoveryServerFactory(f api.ICellFactory, md *api.CellDefinition) (api.IHiveCell, error) {
	httpServer := f.GetHttpServer(true)
	endpoints := make(map[string]string)
	tps := f.GetTransportServers()

	for _, tp := range tps {
		connectURL := tp.GetConnectURL()
		parts := strings.Split(connectURL, ":")
		scheme := parts[0]
		endpoints[scheme] = connectURL
	}
	// Optionally serve the directory TDD if found. See also ServeDirectoryTD()
	tddJSON := ""
	dirSvc, found := f.GetCell(directory.DirectoryServiceCellType).(directory.IDirectoryService)
	if found {
		_, tddJSON = dirSvc.GetTDD()
	}
	serviceName, _ := os.Hostname() // use default
	srv := NewDiscoveryServer(serviceName, httpServer, tddJSON, endpoints)

	return srv, nil
}

package discovery_server

import (
	"strings"

	"github.com/grandcat/zeroconf"
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/cells/directory"
	"github.com/hiveot/hivekit/go/cells/transport/discovery"
	"github.com/hiveot/hivekit/go/cells/transport/discovery/server/internal"
)

// NewDiscoveryServer returns a ready-to-use discovery server instance.
//
// The optional instanceName must be unique for all discovery records as it
// is used in the download URL. Use "" for the default instance.
//
//	httpServer is the server that serves the TD on the well-known endpoint.
//	tdd is the optional directory TD to serve. nil to wait for ServeDirectoryTD()
//	endpoints are optional additional URLS to include in the DNS-SD discovery record
//		 where key is the schema "http", "wss", "sse-sc" and value the URL.
func NewDiscoveryServer(
	httpServer api.IHttpServer, tdd *td.TD, endpoints map[string]string) (discovery.IDiscoveryServer, error) {

	return internal.NewDiscoveryServerImpl(httpServer, tdd, endpoints)
}

// Return a ready-to-use discovery server using the factory environment.
//
// When used in a cell chain together with a directory, this service must be placed
// behind the directory in the chain, so createTD requests from services will be
// handled by the directory and not be served by discovery.
//
// This loads the http server and get a list of server endpoints from the factory, to include
// in discovery.
//
// If a directory cell is available in the factory, its TDD will be retrieved and served as a
// directory record using the "{hostname}:directory" service name.
func NewDiscoveryServerFactory(
	f api.ICellFactory, md *api.CellDefinition) (api.IHiveCell, error) {

	httpServer := f.GetHttpServer(true)
	endpoints := make(map[string]string)
	tps := f.GetTransportServers()

	for _, tp := range tps {
		connectURL := tp.GetConnectURL()
		parts := strings.Split(connectURL, ":")
		scheme := parts[0]
		endpoints[scheme] = connectURL
	}
	// Optionally serve the directory TD if found. See also ServeDirectoryTD()
	var tdd *td.TD
	dirSvc, found := f.GetCell(directory.DirectoryServiceCellType).(directory.IDirectoryService)
	if found {
		tdd = dirSvc.GetTDD()
		f.AddTDSecForms(tdd, true)
	}
	return NewDiscoveryServer(httpServer, tdd, endpoints)
}

// for testing
func ServeDnsSD(instanceName string, subType string, serviceType string,
	address string, port int, params map[string]string) (*zeroconf.Server, error) {
	return internal.ServeDnsSD(instanceName, subType, serviceType, address, port, params)
}

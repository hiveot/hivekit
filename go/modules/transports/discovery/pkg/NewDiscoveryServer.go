package discoverypkg

import (
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/modules/transports/discovery"
	internalserver "github.com/hiveot/hivekit/go/modules/transports/discovery/internal/server"
)

// NewDiscoveryServer creates a new discovery server module instance.
//
//		httpServer is the server that serves the TD on the well-known endpoint.
//		endpoints are optional additional URLS to include in the DNS-SD discovery record
//		 where key is the schema "http", "wss", "sse-sc" and value the URL.
//	 serviceID to publish as. This is the module thingID
func NewDiscoveryServer(serviceID string,
	httpServer transports.IHttpServer, endpoints map[string]string) discovery.IDiscoveryServer {

	srv := internalserver.NewDiscoveryServer(serviceID, httpServer, endpoints)
	return srv
}

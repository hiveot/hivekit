package discovery

import (
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

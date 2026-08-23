package directory_service

import (
	"time"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/cells/directory"
	internal "github.com/hiveot/hivekit/go/cells/directory/internal/httpserver"
)

// Create a new instance
func NewDirectoryHttpServer(httpServer api.IHttpServer, respTimeout time.Duration) directory.IDirectoryHttpServer {
	m := internal.NewDirectoryHttpServer(httpServer, respTimeout)
	return m
}

// Factory for the directory http interface cell
// Place this before the directory service in the chain and before middleware cells that log and
// authorize requests.
func NewDirectoryHttpServerFactory(f api.ICellFactory) api.IHiveCell {

	rpcTimeout := f.GetEnvironment().RpcTimeout
	httpServer, ok := f.GetCell(api.HttpServerCellType).(api.IHttpServer)
	_ = ok
	m := internal.NewDirectoryHttpServer(httpServer, rpcTimeout)
	return m
}

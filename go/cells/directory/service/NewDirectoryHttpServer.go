package directory_service

import (
	"time"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/cells/directory"
	internal "github.com/hiveot/hivekit/go/cells/directory/internal/httpserver"
)

// Returns a ready-to-use directory http handler with the given http server.
func NewDirectoryHttpServer(
	httpServer api.IHttpServer, respTimeout time.Duration) (
	directory.IDirectoryHttpServer, error) {

	return internal.NewDirectoryHttpServer(httpServer, respTimeout)
}

// Factory for the directory http interface cell
// Place this before the directory service in the chain and before middleware cells that log and
// authorize requests.
func NewDirectoryHttpServerFactory(f api.ICellFactory) (api.IHiveCell, error) {

	rpcTimeout := f.GetEnvironment().RpcTimeout
	httpServer, ok := f.GetCell(api.HttpServerCellType).(api.IHttpServer)
	_ = ok
	return internal.NewDirectoryHttpServer(httpServer, rpcTimeout)
}

package directory_service

import (
	"time"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/cells/directory"
	"github.com/hiveot/hivekit/go/cells/directory/service/internal/httpserverimpl"
)

// Returns a ready-to-use directory http handler with the given http server.
func NewDirectoryHttpServer(
	dirThingID string, httpServer api.IHttpServer, respTimeout time.Duration) (
	directory.IDirectoryHttpServer, error) {

	return httpserverimpl.NewDirectoryHttpServerImpl(dirThingID, httpServer, respTimeout)
}

// Factory for the directory http interface cell
// Place this before the directory service in the chain and before middleware cells that log and
// authorize requests.
//
// A directory cell type must be registered in the factory to obtain its thingID.
func NewDirectoryHttpServerFactory(f api.ICellFactory) (api.IHiveCell, error) {

	// Need to know who to forward directory requests to.
	dirSvc := f.GetCell(directory.DirectoryServiceCellType)
	dirThingID := dirSvc.GetID()

	rpcTimeout := f.GetEnvironment().RpcTimeout
	httpServer, ok := f.GetCell(api.HttpServerCellType).(api.IHttpServer)
	_ = ok
	return NewDirectoryHttpServer(dirThingID, httpServer, rpcTimeout)
}

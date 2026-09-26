package rcrouter_service

import (
	"fmt"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/cells/directory"
	"github.com/hiveot/hivekit/go/cells/rcrouter"
	"github.com/hiveot/hivekit/go/cells/rcrouter/service/internal"
)

// NewRCRouterService creates a ready-to-use instance of the reverse connection
// router service.
// Start must be called before usage.
//
//	getTD  handler to lookup a TD for a thingID from a directory. Required.
//	getSrv handler to return the running list of transport servers that can contain
//	 reverse connections. nil to not support RCs.
func NewRCRouterService(
	getTD func(thingID string) *td.TD,
	getSrv func() []api.ITransportServer,
) (rcrouter.IRCRouterService, error) {

	return internal.NewRCRouterServiceImpl(getTD, getSrv)
}

// Create a reverse-connection router service instance using the factory environment.
func NewRCRouterServiceFactory(f api.ICellFactory, md *api.CellDefinition) (api.IHiveCell, error) {

	var getTD func(string) *td.TD

	// The RCrouter needs a directory.
	m, err := f.NewCell(directory.DirectoryServiceCellType, true)
	if err == nil {
		if dirMod, ok := m.(directory.IDirectoryService); ok {
			getTD = dirMod.GetTD
		}
	} else {
		// maybe directory client if the directory lives elsewhere.
		m, err = f.NewCell(directory.DirectoryClientCellType, true)
		if err == nil {
			if dirMod, ok := m.(directory.IDirectoryClient); ok {
				getTD = dirMod.GetTD
			}
		}
	}
	if err != nil {
		return nil, fmt.Errorf("NewRCRouterServiceFactory. Missing directory client or service.")
	}

	svc, err := NewRCRouterService(
		getTD,
		f.GetTransportServers,
	)

	return svc, err
}

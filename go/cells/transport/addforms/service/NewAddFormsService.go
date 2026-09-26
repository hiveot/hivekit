package addforms_service

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/cells/transport/addforms/service/internal"
)

// AddFormsService intercepts and modifies TD's written to a directory.
// The TD is updated with base, security, and form information from the configured
// transport servers.
//
// It is intended to be placed behind the exposed thing in a stand-alone device.
func NewAddFormsServiceFactory(f api.ICellFactory, md *api.CellDefinition) (api.IHiveCell, error) {
	svc := internal.NewAddFormsServiceImpl(f.GetTransportServers)
	return svc, nil
}

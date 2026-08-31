package addforms_service

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/cells/transport/addforms/internal"
)

// AddFormsService intercepts and modifies TD's written to the directory.
// The TD is updated with base, security, and form information from the configured
// transport servers.
// It should be placed behind the publisher in the chain and either before the discovery
// or the directory server, whichever one is used.
// func NewAddFormsService(tpServers []api.ITransportServer) addforms.IAddFormsService {
// 	return internal.NewAddFormsServiceImpl(tpServers)
// }

func StartAddFormsServiceFactory(f api.ICellFactory, md *api.CellDefinition) (api.IHiveCell, error) {
	svc := internal.StartAddFormsServiceImpl(f.GetTransportServers)
	return svc, nil
}

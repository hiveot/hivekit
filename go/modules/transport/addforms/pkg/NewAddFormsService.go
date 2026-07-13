package addformspkg

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/modules/transport/addforms/internal"
)

// AddFormsService intercepts and modifies TD's written to the directory.
// The TD is updated with base, security, and form information from the configured
// transport servers.
// It should be placed behind the publisher in the chain and either before the discovery
// or the directory server, whichever one is used.
// func NewAddFormsService(tpServers []api.ITransportServer) addforms.IAddFormsService {
// 	return internal.NewAddFormsServiceImpl(tpServers)
// }

func NewAddFormsServiceFactory(f api.IModuleFactory, md *api.ModuleDefinition) (api.IHiveModule, error) {
	m := internal.NewAddFormsServiceImpl(f.GetTransportServers)
	return m, nil
}

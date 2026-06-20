package addformspkg

import (
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/factory"
	"github.com/hiveot/hivekit/go/modules/transport"
	"github.com/hiveot/hivekit/go/modules/transport/addforms"
	"github.com/hiveot/hivekit/go/modules/transport/addforms/internal"
)

// AddFormsService intercepts and modifies TD's written to the directory.
// The TD is updated with base, security, and form information from the configured
// transport servers.
// It should be placed behind the publisher in the chain and either before the discovery
// or the directory server, whichever one is used.
func NewAddFormsService(tpServers []transport.ITransportServer) addforms.IAddFormsService {
	return internal.NewAddFormsService(tpServers)
}

func NewAddFormsServiceFactory(f factory.IModuleFactory) (modules.IHiveModule, error) {
	tpServers := f.GetTransportServers()
	m := internal.NewAddFormsService(tpServers)
	return m, nil
}

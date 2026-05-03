package addformspkg

import (
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/factory"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/modules/transports/addforms"
	"github.com/hiveot/hivekit/go/modules/transports/addforms/internal"
)

func NewAddFormsService(tpServers []transports.ITransportServer) addforms.IAddFormsService {
	return internal.NewAddFormsService(tpServers)
}

func NewAddFormsServiceFactory(f factory.IModuleFactory) (modules.IHiveModule, error) {
	tpServers := f.GetTransportServers()
	m := internal.NewAddFormsService(tpServers)
	return m, nil
}

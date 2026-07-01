package factory_test

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/modules/consumer"
	"github.com/hiveot/hivekit/go/modules/thing"
	tlsserverpkg "github.com/hiveot/hivekit/go/modules/transport/tlsserver/pkg"
	wss "github.com/hiveot/hivekit/go/modules/transport/wss"
	wsspkg "github.com/hiveot/hivekit/go/modules/transport/wss/pkg"
)

// module types of a device server device chain
var DeviceServerRecipe = []api.ModuleDefinition{
	{
		Type:        api.HttpServerModuleType,
		Constructor: tlsserverpkg.NewTLSServerFactory,
	},
	{
		Type:        wss.WotWebsocketServerModuleType,
		Constructor: wsspkg.NewWotWssServerFactory,
	},
	{
		Type:        thing.ExposedThingModuleType,
		Constructor: thing.NewExposedThingFactory,
	},
}

// module types of a client chain
var DeviceClientRecipe = []api.ModuleDefinition{
	{
		Type:        consumer.ConsumerModuleType,
		Constructor: consumer.NewConsumerFactory,
	},
	{
		Type:        wss.WotWebsocketClientModuleType,
		Constructor: wsspkg.NewWotWssClientFactory,
	},
}

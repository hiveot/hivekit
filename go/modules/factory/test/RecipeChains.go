package factory_test

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/modules/consumer"
	"github.com/hiveot/hivekit/go/modules/thing"
	tls_server "github.com/hiveot/hivekit/go/modules/transport/tlsserver/server"
	wss "github.com/hiveot/hivekit/go/modules/transport/wss"
	wss_client "github.com/hiveot/hivekit/go/modules/transport/wss/client"
	wss_server "github.com/hiveot/hivekit/go/modules/transport/wss/server"
)

// Recipe chain of a device server device chain
var DeviceServerRecipe = []api.ModuleDefinition{
	{
		Type:        api.HttpServerModuleType,
		Constructor: tls_server.NewTLSServerFactory,
	},
	{
		Type:        wss.WotWebsocketServerModuleType,
		Constructor: wss_server.NewWotWssServerFactory,
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
		Constructor: wss_client.NewWotWssClientFactory,
	},
}

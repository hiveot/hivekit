package factory_test

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/cells/consumer"
	"github.com/hiveot/hivekit/go/cells/thing"
	tls_server "github.com/hiveot/hivekit/go/cells/transport/tlsserver/server"
	wss "github.com/hiveot/hivekit/go/cells/transport/wss"
	wss_client "github.com/hiveot/hivekit/go/cells/transport/wss/client"
	wss_server "github.com/hiveot/hivekit/go/cells/transport/wss/server"
)

// Recipe chain of a device server device chain
var DeviceServerRecipe = []api.CellDefinition{
	{
		Type:        api.HttpServerCellType,
		Constructor: tls_server.NewTLSServerFactory,
	},
	{
		Type:        wss.WotWebsocketServerCellType,
		Constructor: wss_server.NewWotWssServerFactory,
	},
	{
		Type:        thing.ExposedThingCellType,
		Constructor: thing.NewExposedThingFactory,
	},
}

// cell types of a client chain
var DeviceClientRecipe = []api.CellDefinition{
	{
		Type:        consumer.ConsumerCellType,
		Constructor: consumer.NewConsumerFactory,
	},
	{
		Type:        wss.WotWebsocketClientCellType,
		Constructor: wss_client.NewWotWssClientFactory,
	},
}

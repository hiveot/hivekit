package factoryrecipe

import (
	"github.com/hiveot/hivekit/go/modules/clients"
	"github.com/hiveot/hivekit/go/modules/factory"
	"github.com/hiveot/hivekit/go/modules/logging"
	"github.com/hiveot/hivekit/go/modules/transports/wss1"
	wsspkg "github.com/hiveot/hivekit/go/modules/transports/wss1/pkg"
)

// Recipe for a simple device that uses a reverse connection to a gateway
// This creates an application with the following modules:
// 1. websocket client
// 2. logger
// 3. agent for serving device requests
var DeviceRCRecipe = FactoryRecipe{
	ModuleDefs: map[string]factory.ModuleDefinition{
		wss.HiveotWebsocketClientModuleType: {
			Constructor: wsspkg.NewHiveotWssClientFactory,
		},
	},
	ModuleChain: []string{
		wss.HiveotWebsocketClientModuleType,
		logging.LoggingModuleType,
		clients.AgentModuleType,
	},
}

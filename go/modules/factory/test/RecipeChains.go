package factory_test

import (
	"github.com/hiveot/hivekit/go/modules/agent"
	"github.com/hiveot/hivekit/go/modules/consumer"
	"github.com/hiveot/hivekit/go/modules/factory"
	wss "github.com/hiveot/hivekit/go/modules/transport/wss"
	wsspkg "github.com/hiveot/hivekit/go/modules/transport/wss/pkg"
)

// module types of a device server agent chain
var TestDeviceServerRecipe = []factory.ModuleDefinition{
	{
		Type:        wss.WotWebsocketServerModuleType,
		Constructor: wsspkg.NewWotWssServerFactory,
	},
	{
		Type:        agent.AgentModuleType,
		Constructor: agent.NewAgentFactory,
	},
}

// module types of a client chain
var TestDeviceClientRecipe = []factory.ModuleDefinition{
	{
		Type:        consumer.ConsumerModuleType,
		Constructor: consumer.NewConsumerFactory,
	},
	{
		Type:        wss.WotWebsocketClientModuleType,
		Constructor: wsspkg.NewWotWssServerFactory,
	},
}

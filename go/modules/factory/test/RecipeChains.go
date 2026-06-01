package factory_test

import (
	"github.com/hiveot/hivekit/go/modules/agent"
	"github.com/hiveot/hivekit/go/modules/consumer"
	wss "github.com/hiveot/hivekit/go/modules/transport/wss"
)

// module types of a device server agent chain
var TestDeviceServerChain = []string{
	wss.WotWebsocketServerModuleType,
	agent.AgentModuleType,
}

// module types of a client chain
var TestDeviceClientChain = []string{
	consumer.ConsumerModuleType,
	wss.WotWebsocketClientModuleType,
}

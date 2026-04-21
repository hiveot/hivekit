package factory_test

import (
	"github.com/hiveot/hivekit/go/modules/clients"
	"github.com/hiveot/hivekit/go/modules/transports/wss1"
)

// module types of a device server agent chain
var TestDeviceServerChain = []string{
	wss.WotWebsocketServerModuleType,
	clients.AgentModuleType,
}

// module types of a client chain
var TestDeviceClientChain = []string{
	clients.ConsumerModuleType,
	wss.WotWebsocketClientModuleType,
}

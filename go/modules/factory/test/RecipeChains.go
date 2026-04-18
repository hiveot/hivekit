package factory_test

import (
	"github.com/hiveot/hivekit/go/modules/clients"
	"github.com/hiveot/hivekit/go/modules/transports/wss"
)

// module types of a device server agent chain
var TestDeviceServerChain = []string{
	wss.WotWebsocketServerType,
	clients.AgentModuleType,
}

// module types of a device client chain
var TestDeviceClientChain = []string{
	clients.ConsumerModuleType,
	wss.WotWebsocketClientType,
}

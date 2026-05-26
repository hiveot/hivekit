package factory_test

import (
	clientspkg "github.com/hiveot/hivekit/go/modules/transport/clients/pkg"
	wss "github.com/hiveot/hivekit/go/modules/transport/wss"
)

// module types of a device server agent chain
var TestDeviceServerChain = []string{
	wss.WotWebsocketServerModuleType,
	clientspkg.AgentModuleType,
}

// module types of a client chain
var TestDeviceClientChain = []string{
	clientspkg.ConsumerModuleType,
	wss.WotWebsocketClientModuleType,
}

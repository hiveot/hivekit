package factory_test

import (
	clientspkg "github.com/hiveot/hivekit/go/modules/clients/pkg"
	wss "github.com/hiveot/hivekit/go/modules/transports/wss"
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

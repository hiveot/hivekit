package wss

import "github.com/hiveot/hivekit/go/modules/transports"

const (
	// Hiveot websocket sub-protocol
	HiveotWebsocketClientModuleType = "hiveot-wss-client"
	HiveotWebsocketServerModuleType = "hiveot-wss-server"
	HiveotWebsocketServerThingID    = HiveotWebsocketServerModuleType
	HiveotWebsocketPath             = "/hiveot/wss"

	// WoT websocket sub-protocol
	WotWebsocketClientType    = "wot-wss-client"
	WotWebsocketServerType    = "wot-wss-server"
	WotWebsocketServerThingID = WotWebsocketServerType
	WotWebsocketPath          = "/wot/wss"
)

// Interface of the Hiveot websocket server module
type IWssTransportServer interface {
	transports.ITransportServer

	// todo: future API  for servicing the module
}

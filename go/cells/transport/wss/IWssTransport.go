package wss

import "github.com/hiveot/hivekit/go/api"

const (
	// Hiveot websocket sub-protocol
	HiveotWebsocketClientCellType = "hiveot-wss-client"
	HiveotWebsocketServerCellType = "hiveot-wss-server"
	HiveotWebsocketServerThingID  = HiveotWebsocketServerCellType
	HiveotWebsocketPath           = "/hiveot/wss"

	// WoT websocket sub-protocol
	WotWebsocketClientCellType = "wot-wss-client"
	WotWebsocketServerCellType = "wot-wss-server"
	WotWebsocketServerThingID  = WotWebsocketServerCellType
	WotWebsocketPath           = "/wot/wss"
)

// Interface of the Hiveot websocket server
type IWssTransportServer interface {
	api.ITransportServer
}

package wssapi

import "github.com/hiveot/hivekit/go/modules/transports"

const (
	// Hiveot websocket sub-protocol
	HiveotWebsocketModuleID = "hiveot-wss"
	HiveotWebsocketPath     = "/hiveot/wss"

	// WoT websocket sub-protocol
	WotWebsocketModuleID = "wot-wss"
	WotWebsocketPath     = "/wot/wss"
)

// Interface of the Hiveot websocket server module
type IWssTransportServer interface {
	transports.ITransportServer

	// todo: future API  for servicing the module
}

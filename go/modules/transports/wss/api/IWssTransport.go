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

// IWssTransport defines the interface of the Websocket service server
// Used for both WoT and Hiveot websocket message format.
type IWssTransport interface {
	transports.ITransportServer
}

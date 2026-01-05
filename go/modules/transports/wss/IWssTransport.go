package wss

import "github.com/hiveot/hivekit/go/modules/transports"

const (
	DefaultWotWssPath        = "/wot/wss"
	DefaultHiveotWssPath     = "/hiveot/wss"
	SubprotocolHiveotWSS     = "hiveot-wss" // what to use here?
	SubprotocolWotWSS        = "websocket"
	WotWssSchema             = "wss"
	HiveotWssSchema          = "wss"
	DefaultHiveotWssModuleID = "hiveot-wss"
	DefaultWotWssModuleID    = "wot-wss"
)

// IWssTransport defines the interface of the Websocket service server
// Used for both WoT and Hiveot websocket message format.
type IWssTransport interface {
	transports.ITransportModule
}

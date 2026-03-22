package clients

import (
	"crypto/x509"
	"fmt"
	"net/url"
	"strings"

	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/transports"
	httpbasicclient "github.com/hiveot/hivekit/go/modules/transports/httpbasic/client"
	ssescclient "github.com/hiveot/hivekit/go/modules/transports/ssesc/client"
	wssclient "github.com/hiveot/hivekit/go/modules/transports/wss/client"
	"github.com/hiveot/hivekit/go/wot"
	"github.com/hiveot/hivekit/go/wot/td"
)

// IClientModule is the combined interface of a client connection and HiveKit Module.
// This is intended to be used as a sink for publishing requests to a remote server and
// to register a callback for notifications.
type IClientModule interface {
	transports.IClientConnection
	modules.IHiveModule
}

// GetProtocolType returns the protocol used for connecting to this device.
// This returns the protocol type and connection href, if available.
//
// Note that TDs can use multiple protocols for its operations. HiveOT currently assumes
// that only a single protocol is used for connecting with a device. Steps:
//
//  1. lookup the forms of the 'observeallproperties' or 'subscribeallevents' top level operation,
//     if present take the first form and lookup the subprotocol and href fields.
//     if no subprotocol field exists, then use the scheme in the href.
//
//  2. if no form was found then use the scheme of the 'base' field in the td.
//     wss maps to websocket, subprotocol can be longpoll, sse is hiveot sse-sc, etc.
//
// 3. if all else fails if still no scheme then assume its http basic
func GetProtocolType(tdoc *td.TD) (protocolType string, href string) {
	forms := tdoc.GetForms(wot.OpObserveAllProperties, "")
	if len(forms) == 0 {
		forms = tdoc.GetForms(wot.OpSubscribeAllEvents, "")
	}
	if len(forms) == 0 {
		forms = tdoc.Forms // pick any
	}
	if len(forms) > 0 {
		subprotocol, found := forms[0].GetSubprotocol()
		href = forms[0].GetHRef()
		parts, err := url.Parse(href)
		// if href has no scheme then join with the 'base' field
		if err == nil && parts.Scheme == "" && tdoc.Base != "" {
			href, err = url.JoinPath(tdoc.Base, href)
		}
		if found {
			switch subprotocol {
			case transports.HiveotSseScSubprotocol:
				protocolType = transports.HiveotSseScProtocolType
			case transports.HiveotWebsocketSubprotocol:
				protocolType = transports.HiveotWebsocketProtocolType
			case transports.WotWebsocketSubprotocol:
				protocolType = transports.WotWebsocketProtocolType
			case transports.WotHttpLongPollSubprotocol:
				protocolType = transports.WotHttpLongPollProtocolType
			}
		}
	}
	if protocolType != "" {
		return protocolType, href
	}
	// if no subprotocol was found determine it from the href
	if href == "" {
		href = tdoc.Base
	}
	if strings.HasPrefix(href, transports.WotHttpBasicUrlScheme) {
		return transports.WotHttpBasicProtocolType, href
	}
	// a normal TD device should have a subprotocol so not sure what is going on here.
	// just some fallback options
	if strings.HasPrefix(href, transports.WotWebsocketUrlScheme) {
		return transports.WotWebsocketProtocolType, href
	}
	if strings.HasPrefix(href, transports.WotMqttUrlScheme) {
		return transports.WotMqttProtocolType, href
	}
	if strings.HasPrefix(href, transports.WotSseUrlScheme) {
		return transports.WotSseProtocolType, href
	}
	return "", href
}

// NewTransportClient returns a new client module instance ready to connect to a transport server
// using the given URL.
//
// protocolType provides direct control of the client to create regardless of the URL.
// If omitted, then it is derived from the serverURL scheme.
//
// # Use SetTimeout for increasing the default communication timeout for testing
//
// This is intended to be used as a sink for application modules.
func NewTransportClient(protocolType string, serverURL string, caCert *x509.Certificate,
	ch transports.ConnectionHandler) (cl IClientModule, err error) {

	parts, err := url.Parse(serverURL)
	scheme := strings.ToLower(parts.Scheme)
	// the protocol to use is based on scheme

	// use the URL to determine the protocol
	if protocolType == "" {
		if strings.HasPrefix(serverURL, transports.WotWebsocketUrlScheme) {
			protocolType = transports.WotWebsocketProtocolType
		} else if strings.HasPrefix(serverURL, transports.WotSseUrlScheme) {
			protocolType = transports.WotSseProtocolType
		} else if strings.HasPrefix(serverURL, transports.WotMqttUrlScheme) {
			protocolType = transports.WotMqttProtocolType
		} else if strings.HasPrefix(serverURL, transports.WotHttpBasicUrlScheme) {
			protocolType = transports.WotHttpBasicProtocolType
		} else if strings.HasPrefix(serverURL, transports.HiveotSseScUrlScheme) {
			protocolType = transports.HiveotSseScProtocolType
		}
	}

	switch protocolType {
	case transports.HiveotSseScProtocolType:
		cl = ssescclient.NewSseScClient(serverURL, caCert, ch)

	case transports.HiveotWebsocketProtocolType:
		cl = wssclient.NewHiveotWssClient(serverURL, caCert, ch)

	case transports.WotWebsocketProtocolType:
		cl = wssclient.NewWotWssClient(serverURL, caCert, ch)

	case transports.WotHttpBasicProtocolType:
		caCert := caCert
		cl = httpbasicclient.NewHttpBasicClient(serverURL, caCert, nil, ch)

	//case transports.ProtocolTypeWotMQTTWSS:
	//	fullURL = testServerMqttWssURL
	default:
		err = fmt.Errorf("NewTransportClient. Unknown protocol '%s'", scheme)
	}
	return cl, err
}

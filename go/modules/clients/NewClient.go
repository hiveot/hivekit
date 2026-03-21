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
	"github.com/hiveot/hivekit/go/wot/td"
)

// IClientModule is the combined interface of a client connection and HiveKit Module.
// This is intended to be used as a sink for publishing requests to a remote server and
// to register a callback for notifications.
type IClientModule interface {
	transports.IClientConnection
	modules.IHiveModule
}

// NewTransportClient returns a new client module instance ready to connect to a transport server
// using the given URL.
//
// protocolType provides direct control of the client to create regardless of the URL.
// If omitted, then it is derived from the serverURL scheme.
//
// This is intended to be used as a sink for application modules.
func NewTransportClient(protocolType string, serverURL string, caCert *x509.Certificate,
	ch transports.ConnectionHandler) (cl IClientModule, err error) {

	parts, err := url.Parse(serverURL)
	scheme := strings.ToLower(parts.Scheme)
	// the protocol to use is based on scheme

	// use the URL to determine the protocol
	if protocolType == "" {
		if strings.HasPrefix(serverURL, td.ProtocolSchemeWotWSS) {
			protocolType = td.ProtocolTypeWotWSS
		} else if strings.HasPrefix(serverURL, td.ProtocolSchemeWotSSE) {
			protocolType = td.ProtocolTypeWotSSE
		} else if strings.HasPrefix(serverURL, td.ProtocolSchemeWotMQTTWSS) {
			protocolType = td.ProtocolTypeWotMQTTWSS
		} else if strings.HasPrefix(serverURL, td.ProtocolSchemeHTTPBasic) {
			protocolType = td.ProtocolTypeHTTPBasic
		} else if strings.HasPrefix(serverURL, td.ProtocolSchemeHiveotSSESC) {
			protocolType = td.ProtocolTypeHiveotSSESC
		}
	}

	switch protocolType {
	case td.ProtocolTypeHiveotSSESC: // "sse"
		cl = ssescclient.NewSseScClient(serverURL, caCert, ch)

	case td.ProtocolTypeHiveotWSS: // "hiveot-wss"
		cl = wssclient.NewHiveotWssClient(serverURL, caCert, 0, ch)

	case td.ProtocolTypeWotWSS: // "websocket"
		cl = wssclient.NewWotWssClient(serverURL, caCert, 0, ch)

	case td.ProtocolTypeHTTPBasic: // "https"
		caCert := caCert
		cl = httpbasicclient.NewHttpBasicClient(serverURL, caCert, nil, ch)

	//case transports.ProtocolTypeWotMQTTWSS:
	//	fullURL = testServerMqttWssURL
	default:
		err = fmt.Errorf("NewTransportClient. Unknown protocol '%s'", scheme)
	}
	return cl, err
}

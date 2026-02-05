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
)

// IClientModule is the combined interface of a client connection and HiveKit Module.
// This is intended to be used as a sink for publishing requests to a remote server and
// to register a callback for notifications.
type IClientModule interface {
	transports.IConnection
	modules.IHiveModule
}

// NewTransportClient returns a new client module instance ready to connect to a transport server
// using the given URL.
// This is intended to be used as a sink for application modules.
func NewTransportClient(serverURL string, caCert *x509.Certificate) (cl IClientModule, err error) {
	parts, err := url.Parse(serverURL)
	scheme := strings.ToLower(parts.Scheme)

	switch scheme {
	case transports.ProtocolTypeHiveotSSE: // "sse"
		cl = ssescclient.NewSseScClient(serverURL, caCert)

	case transports.ProtocolTypeWotWSS: // "wss"
		cl = wssclient.NewWotWssClient(serverURL, caCert)

	case transports.ProtocolTypeHTTPBasic: // "https"
		caCert := caCert
		cl = httpbasicclient.NewHttpBasicClient(serverURL, caCert, nil)

	//case transports.ProtocolTypeWotMQTTWSS:
	//	fullURL = testServerMqttWssURL

	default:
		err = fmt.Errorf("NewTransportClient. Unknown protocol '%s'", scheme)
	}
	return cl, err
}

package clients

import (
	"crypto/x509"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/transports"
	httpbasicclient "github.com/hiveot/hivekit/go/modules/transports/httpbasic/client"
	ssescclient "github.com/hiveot/hivekit/go/modules/transports/ssesc/client"
	wssclient "github.com/hiveot/hivekit/go/modules/transports/wss/client"
)

// IClientModule is the combined interface of a client connection and HiveKit Module
// This is intended to be used as a sink for publishing requests to a remote server.
type IClientModule interface {
	transports.IConnection
	modules.IHiveModule
}

// NewClientModule returns a new client instance ready to connect and be used as a sink
func NewClientModule(serverURL string, caCert *x509.Certificate, timeout time.Duration) (cl IClientModule, err error) {
	parts, err := url.Parse(serverURL)
	scheme := strings.ToLower(parts.Scheme)

	switch scheme {
	case transports.ProtocolTypeHiveotSSE: // "sse"
		cl = ssescclient.NewSseScClient(serverURL, caCert, timeout)

	case transports.ProtocolTypeWotWSS: // "wss"
		cl = wssclient.NewWotWssClient(serverURL, caCert, timeout)

	case transports.ProtocolTypeHTTPBasic: // "https"
		caCert := caCert
		cl = httpbasicclient.NewHttpBasicClient(
			serverURL, caCert, nil, timeout)

	//case transports.ProtocolTypeWotMQTTWSS:
	//	fullURL = testServerMqttWssURL

	default:
		err = fmt.Errorf("NewClient. Unknown protocol '%s'", scheme)
	}
	return cl, err
}

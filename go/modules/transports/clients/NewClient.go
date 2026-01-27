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

type IClientSink interface {
	transports.IConnection
	modules.IHiveModule
}

// NewClientSink returns a new client instance ready to connect and be used as a sink
func NewClientSink(serverURL string, caCert *x509.Certificate, timeout time.Duration) (cl IClientSink, err error) {
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

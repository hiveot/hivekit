// package wsstransport with facade to websocket transport client and server
package wsstransport

import (
	"crypto/x509"

	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/modules/transports/wss/internal/wssclient"
)

// NewHiveotWssClient creates a new instance of the hiveot websocket client.
//
// This uses the Hiveot passthrough message converter.
//
//	wssURL is the full websocket connection URL including path
//	caCert is the server CA for TLS connection validation
//	ch is the connect/disconnect callback. nil to ignore
func NewHiveotWssClient(wssURL string, caCert *x509.Certificate,
	ch transports.ConnectionHandler) transports.ITransportClient {
	return wssclient.NewHiveotWssClient(wssURL, caCert, ch)
}

// NewWotWssClient creates a new instance of the WoT compatible websocket client.
//
// messageConverter offers the ability to use any websocket message format that
// can be mapped to a RequestMessage and ResponseMessage. It is used to support
// both hiveot and WoT websocket message formats.
//
// Users must use ConnectWithToken to authenticate and connect.
//
//	wssURL is the full websocket connection URL
//	caCert is the server CA for TLS connection validation
//	timeout is the maximum connection wait time. 0 for default.
//	ch is the connection callback handler, nil to ignore
func NewWotWssClient(wssURL string, caCert *x509.Certificate,
	ch transports.ConnectionHandler) transports.ITransportClient {
	return wssclient.NewWotWssClient(wssURL, caCert, ch)
}

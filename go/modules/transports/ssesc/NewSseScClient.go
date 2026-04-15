package ssesc

import (
	"crypto/x509"

	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/modules/transports/ssesc/internal/client"
)

// NewHiveotSseClient creates a new instance of the hiveot SSE-SC client.
//
//	sseURL is the full websocket connection URL including path
//	caCert is the server CA for TLS connection validation
//	ch is the connect/disconnect callback. nil to ignore
func NewHiveotSseClient(sseURL string, caCert *x509.Certificate,
	ch transports.ConnectionHandler) transports.ITransportClient {

	return client.NewHiveotSseClient(sseURL, caCert, ch)
}

// NewWotSseClient creates a new instance of the WoT compatible SSE client.
//
// Users must use ConnectWithToken to authenticate and connect.
//
//	sseURL is the full SSE server connection URL
//	caCert is the server CA for TLS connection validation
//	timeout is the maximum connection wait time. 0 for default.
//	ch is the connection callback handler, nil to ignore
// func NewWotSseClient(sseURL string, caCert *x509.Certificate,
// 	ch transports.ConnectionHandler) transports.ITransportClient {

// 	return sseclient.NewWotSseClient(sseURL, caCert, ch)
// }

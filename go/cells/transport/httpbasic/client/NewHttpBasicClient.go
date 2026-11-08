package httpbasic_client

import (
	"crypto/x509"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/wot"
	"github.com/hiveot/hivekit/go/cells/transport/httpbasic/client/internal"
)

// NewHttpBasicClient creates a new instance of the WoT compatible http-basic
// protocol binding client.
//
// Users must use SetAuthToken or SetClientCert to authenticate followed by Start()
// or Connect() to establish a connection.
//
// This uses the given TD to determine the URLs to perform an operation.
//
//	tdoc containing the http URLs for operations.
//	caCert of the server to validate the server or nil to not check the server cert
func NewHttpBasicClient(
	tdoc *wot.TD, rootCAs *x509.CertPool) (api.ITransportClient, error) {

	return internal.NewHttpBasicClientImpl(tdoc, rootCAs)
}

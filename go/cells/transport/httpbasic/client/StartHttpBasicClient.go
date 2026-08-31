package httpbasic_client

import (
	"crypto/x509"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/cells/transport/httpbasic/internal/clientimpl"
)

// StartHttpBasicClient creates a new instance of the WoT compatible http-basic
// protocol binding client.
//
// Users must use SetAuthToken or SetClientCert to authenticate.
//
// This uses the given TD to determine the URLs to perform an operation.
//
//	baseURL of the http server. Used as the base for all further requests.
//	caCert of the server to validate the server or nil to not check the server cert
func StartHttpBasicClient(
	tdoc *td.TD, rootCAs *x509.CertPool) (api.ITransportClient, error) {

	return clientimpl.StartHttpBasicClientImpl(tdoc, rootCAs)
}

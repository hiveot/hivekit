package httpbasicpkg

import (
	"crypto/x509"

	"github.com/hiveot/hivekit/go/modules/transports"
	internalclient "github.com/hiveot/hivekit/go/modules/transports/httpbasic/internal/client"
)

// NewHttpBasicClient creates a new instance of the WoT compatible http-basic
// protocol binding client.
//
// Users must use ConnectWithToken to authenticate and connect.
//
// This uses TD forms to perform an operation.
//
//	baseURL of the http server. Used as the base for all further requests.
//	caCert of the server to validate the server or nil to not check the server cert
//	getForm is the handler for return a form for invoking an operation. nil for default
//	ch optional callback with connection status changes
func NewHttpBasicClient(
	baseURL string, caCert *x509.Certificate,
	getForm transports.GetFormHandler,
	ch transports.ConnectionHandler) transports.ITransportClient {

	return internalclient.NewHttpBasicClient(baseURL, caCert, getForm, ch)
}

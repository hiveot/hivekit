package httpbasicpkg

import (
	"crypto/x509"

	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/factory"
	"github.com/hiveot/hivekit/go/modules/transport"
	internalclient "github.com/hiveot/hivekit/go/modules/transport/httpbasic/internal/client"
)

// NewHttpBasicClient creates a new instance of the WoT compatible http-basic
// protocol binding client.
//
// Users must use AuthenticateWithToken to authenticate and Connect to connect.
//
// This uses TD forms to perform an operation.
//
//	baseURL of the http server. Used as the base for all further requests.
//	caCert of the server to validate the server or nil to not check the server cert
//	getForm is the handler for return a form for invoking an operation. nil for default
//	ch optional callback with connection status changes
func NewHttpBasicClient(
	baseURL string, caCert *x509.Certificate,
	getForm transport.GetFormHandler) transport.ITransportClient {

	return internalclient.NewHttpBasicClient(baseURL, caCert, getForm)
}

// Create an HTTP-Basic client using the application environment from the provided factory
func NewHttpBasicClientFactory(f factory.IModuleFactory) modules.IHiveModule {

	env := f.GetEnvironment()
	m := NewHttpBasicClient(env.ServerURL, env.CaCert, nil)
	clientCert, _ := env.GetClientCert()
	if clientCert != nil {
		m.AuthenticateWithClientCert(clientCert)
	}
	m.SetTimeout(env.RpcTimeout)
	return m
}

package httpbasic_client

import (
	"crypto/x509"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/modules/transport/httpbasic/internal/clientimpl"
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
	getForm api.GetFormHandler) api.ITransportClient {

	return clientimpl.NewHttpBasicClientImpl(baseURL, caCert, getForm)
}

// Create an HTTP-Basic client using the application environment from the provided factory
func NewHttpBasicClientFactory(f api.IModuleFactory, md *api.ModuleDefinition) (api.IHiveModule, error) {
	var err error
	env := f.GetEnvironment()
	m := NewHttpBasicClient(env.ServerURL, env.CaCert, nil)
	clientCert, _ := env.GetTLSCert()
	if clientCert != nil {
		err = m.AuthenticateWithClientCert(clientCert)
	}
	m.SetTimeout(env.RpcTimeout)
	return m, err
}

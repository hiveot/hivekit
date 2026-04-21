package httpbasicpkg

import (
	"crypto/x509"

	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/factory"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/modules/transports/httpbasic/internal/client"
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

	return client.NewHttpBasicClient(baseURL, caCert, getForm, ch)
}

// Create an HTTP-Basic client using the application environment from the provided factory
func NewHttpBasicClientFactory(f factory.IModuleFactory) modules.IHiveModule {

	env := f.GetEnvironment()
	m := NewHttpBasicClient(env.ServerURL, env.CaCert, nil, nil)
	clientCert, _ := env.GetClientCert()
	if clientCert != nil {
		m.ConnectWithClientCert(clientCert)
	}
	m.SetTimeout(env.RpcTimeout)
	return m
}

// NewHttpBasicFormClient creates a new instance of the WoT compatible http-basic
// protocol binding client using forms to connect.
//
//	tlsClient used for the server connection
//	getForm is the handler for return a form for invoking an operation. nil for default
//	ch optional callback with connection status changes
func NewHttpBasicFormClient(
	tlsClient transports.ITLSClient, getForm transports.GetFormHandler,
	ch transports.ConnectionHandler) transports.ITransportClient {

	return client.NewHttpBasicTLSClient(tlsClient, getForm, ch)
}

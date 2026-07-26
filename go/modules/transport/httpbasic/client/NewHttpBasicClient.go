package httpbasic_client

import (
	"crypto/x509"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules/transport/httpbasic/internal/clientimpl"
)

// NewHttpBasicClient creates a new instance of the WoT compatible http-basic
// protocol binding client.
//
// Users must use AuthenticateWithToken to authenticate and Connect to connect.
//
// This uses the given TD to connect and perform an operation.
//
//	baseURL of the http server. Used as the base for all further requests.
//	caCert of the server to validate the server or nil to not check the server cert
func NewHttpBasicClient(
	tdoc *td.TD, caCert *x509.Certificate) api.ITransportClient {

	return clientimpl.NewHttpBasicClientImpl(tdoc, caCert)
}

// Create an HTTP-Basic client using the application environment from the provided factory.
// This obtains a directory client from the factory to lookup forms for operations.
// func NewHttpBasicClientFactory(f api.IModuleFactory, md *api.ModuleDefinition) (api.IHiveModule, error) {
// 	var err error
// 	var getThing func(thingID string) *td.TD
// 	env := f.GetEnvironment()

// 	// http basic client needs the TD of the device to connect to. When should this
// 	// be provided?
// 	//  1. first request obtains the TD from directory client.
// 	//  2. first request obtains the TD from getTD callback
// 	//  3. on client instantiation
// 	//  4. on auth
// 	//  5. with md.Config  <-
// 	//            instead of a serverURL can env contain a serverTD?
// 	//  6. should this even have a factory fn? factory instantiates on startup and this cant
// 	// when
// 	// this can also be used to connect to a directory itself as long as the client
// 	// has the tdd cached.
// 	dir := api.GetFactoryModule[directory.IDirectoryClient](f, directory.DirectoryClientModuleType)
// 	if dir != nil {
// 		getThing = dir.Cache().GetThing
// 	}
// 	m := NewHttpBasicClient(env.ServerURL, env.CaCert, getThing)
// 	clientCert, _ := env.GetTLSCert()
// 	if clientCert != nil {
// 		err = m.AuthenticateWithClientCert(clientCert)
// 	}
// 	m.SetTimeout(env.RpcTimeout)
// 	return m, err
// }

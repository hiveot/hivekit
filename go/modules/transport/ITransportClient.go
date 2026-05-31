package transport

import (
	"crypto/tls"

	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules"
)

// Actions implemented in transport clients
const (
	// Ask the client module to connect with previously set credentials.
	// the action responds with the completed or failed result.
	// If Connect is not supported the request should return with an error.
	ClientConnectAction = "connect"
)

// notification that the client connect status has changed.
// the payload is the new connection status.
// The notification thingID is the client's module-id.
// Note that connection status events are never transmitted to and from the server.
const ClientConnectionStatusEvent = "connectionStatus"

// GetCredentials is the handler that provides the credentials for connecting
// to a transport server.
//
// If the TD has no security info, this returns the scheme auto, which means
// that the protocol uses its default authentication scheme.
//
// This returns:
// - clientID is the account on the device to connect to.
// - cred is the credentials to authenticate with
// - credType is the type of credentials stored, eg bearer token, digist, etc
// - error if the destination is unknown.
type GetCredentials func(thingID string) (clientID string, cred string, credType string, err error)

// GetFormHandler is the handler that provides the client with the form needed to invoke an operation
// This returns the form and a full href for the operation. Relative href's are converted
// to full hrefs.
type GetFormHandler func(op string, thingID string, name string) (f *td.Form, href string)

// ITransportClient defines the interface of a transport client connection.
// This implements IHiveModule and IConnection interfaces.
//
// Note that transport clients do not retain subscription status. If a connection drops
// then event subscriptions and property observations have to be re-issued by the application.
// See the 'Reconnect' module that manages automatic reconnection and restoring of subscriptions.
//
// Transport clients issue ClientConnectionStatusEvent notifications when the connection
// status changes.
type ITransportClient interface {
	modules.IHiveModule
	IConnection

	// AuthenticateWithClientCert sets the authentication credentials to the client certificate.
	//
	// The client certificate common name is the client ID and must be signed by the
	// same CA as the server.
	//
	// This returns an error if the certificate is invalid for the current CA, if
	// certificate authentication is not supported or if an existing connection is not closed.
	AuthenticateWithClientCert(clientCert *tls.Certificate) error

	// AuthenticateWithForm determines authentication credentials using forms and the given
	// getCredentials handler.
	//
	// This determines which auth schema the TD describes, obtains the credentials
	// and injects the authentication credentials according to the TDI schema.
	//
	// Use Connect() to establish a connection.
	//
	// This returns an error if credentials cannot be determined or obtained or if an
	// existing connection is not closed.
	AuthenticateWithForm(tdi *td.TD, getCredentials GetCredentials) error

	// AuthenticateWithToken sets the authentication credentials to the given clientID and
	// token.
	//
	// Use Connect() to establish a connection.
	//
	// This method can be used if it is known that token authentication is supported by
	// the server. The method of obtaining a token depends on the application environment.
	// The authn module can be used for token authentication using LoginWithPassword.
	//
	// If the transport client is started by the module factory, credentials can be
	// provided through the included AppEnvironment using client certificate or token,
	// and used when Start() is called to establish a connection. If the AppEnvironment
	// does not contain credentials then AuthenticateWithToken must be used on the client
	// module obtained using factoryInstance.GetModule(TransportClientType) to establish
	// the connection before the chain can be used.
	//
	//	clientID is the ID to authenticate as, it must match the token
	//	token is the authentication token obtained on login
	//
	// This returns an error if token authentication is not supported or if an existing
	// connection is not closed.
	AuthenticateWithToken(clientID, token string) error

	// Connect using the previously set connection credentials.
	//
	// This returns nil on success or an error if failed.
	Connect() (err error)
}

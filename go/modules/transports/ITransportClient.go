package transports

import (
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules"
)

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
// This returns nil if no form is found for the operation.
type GetFormHandler func(op string, thingID string, name string) *td.Form

// ITransportClient defines the interface of a transport client connection.
// This implements IHiveModule and IConnection.
type ITransportClient interface {
	modules.IHiveModule
	IConnection

	// Authenticate the client connection with the server.
	// This determine which auth schema the TD describes, obtains the credentials
	// and injects the authentication credentials according to the TDI schema.
	// This returns an error if the schema isn't supported or is not compatible.
	//
	// Alternatively, use ConnectWithToken if it is known that token authentication is supported.
	Authenticate(tdi *td.TD, getCredentials GetCredentials) error

	// ConnectWithToken connects to the transport server using a clientID and
	// corresponding authentication token.
	//
	// This method can be used if it is known that bearer token basic authentication is supported
	// by the server.
	//
	// While most hiveot transport servers support token authentication, the method
	// of obtaining a token depends on the environment. The authn module is intended for this.
	//
	// If a connection is already established on this client then it will be closed first.
	//
	// This connection method must be supported by all client implementations.
	//
	//	clientID is the ID to authenticate as, it must match the token
	//	token is the authentication token obtained on login
	//
	// This returns an error if the token is not valid
	ConnectWithToken(clientID, token string) (err error)
}

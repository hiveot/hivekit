package api

import (
	"crypto/tls"
)

// Actions implemented in transport clients
const (
	// Experimental: Ask the client to connect with previously set credentials.
	// the action responds with the completed or failed result.
	// If Connect is not supported the request should return with an error.
	ClientConnectAction = "connect"
)

// Experimental: notification that the client connect status has changed.
// the payload is the new connection status.
// The notification thingID is the client's cell-id.
// Note that connection status events are never transmitted to and from the server.
const ClientConnectionStatusEvent = "connectionStatus"

// Connection status values
type ConnectionStatus string

// connection state machine:
//
//	1: new|lost|closed -> connecting -> connected -> closed
//	2: connecting -> connected -> lost
//	3: connecting -> refused
const (
	// no connection attempt has been made
	StatusNew ConnectionStatus = ""

	// establishing the connection is in progress
	// Calling Connect() returns an error
	StatusConnecting ConnectionStatus = "connecting"

	// the connection was successfully established
	// this is the only status that counts as is-connected.
	// Calling Connect() returns as success without any changes.
	StatusConnected ConnectionStatus = "connected"

	// the connection was been closed by the user
	// Connect can be called to re-establish the connection.
	StatusClosed ConnectionStatus = "closed"

	// the connection was dropped or server not reachable
	// Connect can be called to attempt to re-establish the connection.
	StatusLost ConnectionStatus = "lost"

	// the connection was refused due to incorrect authentication.
	// reauthentication is required.
	// Calling Connect will keep failing until the credentials are valid.
	StatusRefused ConnectionStatus = "refused"
)

// GetCredentials is the handler that provides the credentials for connecting to a thing server.
//
// thingID identifies the Thing to connect to. If no credentials are set for the device then
// this attempts to obtain the default credentials, set with thingID "". If none are
// found this returns an error.
//
// If the TD has no security info, this returns the scheme auto, which means
// that the protocol uses its default authentication scheme.
//
// If a device serves multiple things then use the deviceID instead.
// TODO: is it better to use origin to avoid mixup with devices and things?
//
// This returns:
// - clientID is the account on the device to connect to.
// - cred is the secret credentials to authenticate with
// - credType is the type of credentials stored, eg bearer token, digist, etc
// - found is true if credentials for thingID are found
type GetCredentials func(thingID string) (clientID string, cred string, credType string, found bool)

// ITransportClient defines the interface of a transport client connection.
// This implements IHiveCell and IConnection interfaces.
//
// Note that transport clients do not retain subscription status. If a connection drops
// then event subscriptions and property observations have to be re-issued by the application.
// See the 'Reconnect' cell that manages automatic reconnection and restoring of subscriptions.
//
// Transport clients issue ClientConnectionStatusEvent notifications when the connection
// status changes.
type ITransportClient interface {
	IHiveCell
	IConnection

	// Connect using the previously set connection credentials. See AuthenticateWith...
	//
	// If an error is returned then call GetConnectionStatus to determine why and
	// whether to attempt to Connect again. If the status is ConnectRefused then the
	// credentials are invalid.
	//
	// Connect does not restore subscriptions.
	//
	// An error is return if unable to connect for any reason.
	// The error is UnauthorizedError if credentials are invalid.
	Connect() (err error)

	// Return the connecting status
	GetConnectionStatus() ConnectionStatus

	// SetAuthToken sets the authentication credentials to a supported the token based
	// security scheme. See also td.SecScheme...
	//
	// SetAuthToken or SetClientCert is require to connect, even if nosec is used.
	//
	// Use Connect() to establish a connection.
	//
	// If the provided secScheme is not supported by the transport then Connect will
	// return an error.
	//
	//	clientID is the ID to authenticate as, it must match the token. Required.
	//	token is the authentication token obtained on login
	//	secScheme identifies the authentication security scheme to use. Defaults to SecSchemeNoSec.
	//    for example td.SecSchemeBearer, SecSchemeNoSec, ...
	//
	// This returns an error if token authentication is not supported or if an existing
	// connection is not closed.
	SetAuthToken(clientID string, token string, secScheme string) error

	// SetClientCert sets the client certificate for mutual authentication.
	//
	// The client certificate common name is the client ID and must be signed by the
	// same CA as the server.
	//
	// This returns an error if the certificate is invalid for the current CA pool,
	// or if a connection attempt is in progress.
	SetClientCert(clientCert *tls.Certificate) error

	// SetConnectHandler sets the callback handler that is invoked when the connection
	// status changes.
	// Intended for applications to handle reconnect and resubscription.
	SetConnectHandler(h func(newStatus ConnectionStatus, c ITransportClient))
}

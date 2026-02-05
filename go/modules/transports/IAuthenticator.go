package transports

import (
	"time"

	"github.com/hiveot/hivekit/go/wot/td"
)

var UnauthorizedError error = unauthorizedError{}

// ValidateTokenHandler is the handler definition for validating authentication tokens.
// In http this is the bearer token in the authorization header.
// This handler is provided by the authn module, but can also be used by other authentication methods.
//
// This returns an error if the token is invalid, the clientID is unknown or the token has expired.
type ValidateTokenHandler func(token string) (clientID string, role string, validUntil time.Time, err error)

// Predefined roles of a client
// The roles are hierarchical in permissions:
// Authorization using these roles is applied through an authz service.
// Custom roles can be added if needed but their persmissions need to be
// managed in the authz service.
const (

	// ClientRoleNone means that the client has no permissions.
	// It can not do anything until the role is upgraded to viewer or better
	ClientRoleNone string = "none"

	// ClientRoleViewer for users that can view information for devices/services
	// they have access to.
	// Viewers cannot invoke actions or change configuration.
	ClientRoleViewer string = "viewer"

	// ClientRoleAgent for devices and services.
	//
	// Agents publish device information for the devices/services it manages and
	// receive request for those devices/services.
	ClientRoleAgent string = "agent"

	// ClientRoleOperator for users that operate devices and services.
	//
	// Operators can view and control devices/services they have access to but
	// not configure them.
	ClientRoleOperator string = "operator"

	// ClientRoleManager for users that manage devices.
	//
	// Managers can view, control and configure devices/services they have access to.
	ClientRoleManager string = "manager"

	// ClientRoleAdmin for users that administer the system.
	//
	// Administrators can view, control and configure all devices and services.
	ClientRoleAdmin string = "admin"

	// ClientRoleService for Service role
	//
	// Services are equivalent to an admin user and agent for devices/services they
	// have access to.
	ClientRoleService string = "service"
)

// UnauthorizedError for dealing with authorization problems
type unauthorizedError struct {
	Message string
}

func (e unauthorizedError) Error() string {
	return "Unauthorized: " + e.Message
}

// IAuthenticator is the interface of the authentication capability to obtain and
// validate authentication tokens.
type IAuthenticator interface {
	// AddClient adds a client to the user database
	//
	//  role is the role of the client, eg ClientRoleManager
	//  pubKeyPem is option and intended for (future) encryption
	//
	// This returns an error if the client already exists
	AddClient(clientID string, role string, password string, pubKeyPem string) error

	// AddSecurityScheme adds the wot securityscheme to the given TD
	AddSecurityScheme(tdoc *td.TD)

	// CreateToken creates a signed authentication token for a client.
	//
	// The client must be a known client with an assigned role.
	//
	// If no session has started, a new one will be created. This is intended for
	// issuing agent tokens (devices, services) where login is not applicable.
	//
	// Note that accidentally created tokens can be invalidated by invoking Logout.
	// The authenticator tracks a sessionStart time and only tokens created
	// after the sessionStart times are valid.
	//
	//	clientID identifies the client
	//	validity is the duration of the token starting
	//
	// This returns an error if clientID is missing or validity is 0
	CreateToken(clientID string, validity time.Duration) (token string, validUntil time.Time, err error)

	// DecodeToken and return its claims
	DecodeToken(token string, signedNonce string, nonce string) (
		clientID string, role string, issuedAt time.Time, validUntil time.Time, err error)

	// GetAlg returns the supported security format and authentication algorithm.
	// This uses the vocabulary as defined in the TD.
	// JWT: "ES256", "ES512", "EdDSA"
	// paseto: "local" (symmetric), "public" (asymmetric)
	GetAlg() (string, string)

	// Login with a password and obtain a new authentication token with limited duration.
	// The token must be refreshed before it expires.
	//
	// Token validation is determined through configuration.
	//
	// This returns the authentication token and the expiration time before it must be refreshed.
	// If the login fails this returns an error
	Login(login string, password string) (token string, validUntil time.Time, err error)

	// Logout invalidates all tokens of this client issued before now.
	Logout(clientID string)

	// RefreshToken issues a new authentication token with an updated expiry time.
	// This extends the life of the session.
	//
	//	clientID Client whose token to refresh
	//	oldToken must be valid
	//
	// This returns the token and the validity time before it must be refreshed,
	// If the clientID is unknown or oldToken is no longer valid this returns an error
	RefreshToken(clientID string, oldToken string) (newToken string, validUntil time.Time, err error)

	// SetPassword changes a client's password.
	SetPassword(clientID string, password string) error

	// Set the URI where to login
	// SetAuthServerURI(authServiceURI string)

	// ValidatePassword checks if the given password is valid for the client
	ValidatePassword(clientID string, password string) (err error)

	// ValidateToken verifies the token and client are valid.
	// This returns an error if the token is invalid, the token has expired,
	// or the client is not a valid and enabled client.
	ValidateToken(token string) (clientID string, role string, validUntil time.Time, err error)
}

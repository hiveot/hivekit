package transports

import (
	"time"
)

// IAuthValidator is the interface of the client authentication validator.
// This provides the methods for verifying a clients authenticity, login and renew tokens.
//
// See also the authn module that provides authenticators for managing clients.
type IAuthValidator interface {
	// AddSecurityScheme adds the wot securityscheme to the given TD
	// AddSecurityScheme(tdoc *td.TD)

	// CreateToken creates a signed authentication token for a client.
	//
	// If no session has started, a new one will be created. This is intended for
	// issuing agent tokens (devices, services) where login is not applicable.
	//
	// The use of role is a convenience for authorization usage. Note that accidentally
	// created admin role tokens can be invalidated by invoking Logout.
	// The authenticator tracks a sessionStart time and only tokens created
	// after the sessionStart times are valid.
	//
	//	clientID identifies the client
	//	role includes the role this client can fulfil with this token
	//	validity is the duration of the token starting
	// CreateToken(clientID string, role string, validity time.Duration) (token string, validUntil time.Time)

	// DecodeToken and return its claims
	// DecodeToken(token string, signedNonce string, nonce string) (
	// 	clientID string, role string, issuedAt time.Time, validUntil time.Time, err error)

	// GetAlg returns the supported security format and authentication algorithm.
	// This uses the vocabulary as defined in the TD.
	// JWT: "ES256", "ES512", "EdDSA"
	// paseto: "local" (symmetric), "public" (asymmetric)
	// GetAlg() (string, string)

	// Login with a password and obtain a new authentication token with limited duration.
	// The token must be refreshed before it expires.
	//
	// Token validation is determined through configuration.
	//
	// This returns the authentication token and the expiration time before it must be refreshed.
	// If the login fails this returns an error
	// Login(login string, password string) (token string, validUntil time.Time, err error)

	// Logout invalidates all tokens of this client issued before now.
	// Logout(clientID string)

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
	// SetPassword(clientID string, password string) error

	// Set the URI where to login
	// SetAuthServerURI(authServiceURI string)

	// ValidatePassword checks if the given password is valid for the client
	// ValidatePassword(clientID string, password string) (err error)

	// ValidateToken verifies the token and client are valid.
	// This returns an error if the token is invalid, the token has expired,
	// or the client is not a valid and enabled client.
	ValidateToken(token string) (clientID string, role string, validUntil time.Time, err error)
}

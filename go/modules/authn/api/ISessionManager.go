package authnapi

import (
	"time"

	"github.com/hiveot/hivekit/go/modules/transports"
)

// Interface of client session management that also support transport authentication
type ISessionManager interface {
	transports.IAuthenticator

	// DecodeToken decodes the given token using the configured authenticator.
	// DecodeToken(token string, signedNonce string, nonce string) (
	// 	clientID string, issuedAt time.Time, validUntil time.Time, err error)

	// CreateToken creates a signed authentication token for a client.
	//
	// The client must be a known client.
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

	// ValidatePassword checks if the given password is valid for the client
	ValidatePassword(clientID string, password string) (err error)
}

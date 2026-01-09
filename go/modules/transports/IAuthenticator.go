package transports

import (
	"time"

	"github.com/hiveot/hivekit/go/wot/td"
)

var UnauthorizedError error = unauthorizedError{}

// UnauthorizedError for dealing with authorization problems
type unauthorizedError struct {
	Message string
}

func (e unauthorizedError) Error() string {
	return "Unauthorized: " + e.Message
}

// IAuthenticator is the interface of the authentication capability to obtain and
// validate session tokens.
type IAuthenticator interface {
	// AddSecurityScheme adds the wot securityscheme to the given TD
	AddSecurityScheme(tdoc *td.TD)

	// CreateSessionToken creates a signed session token for a client and adds the session
	// sessionID is required. For persistent sessions use the clientID.
	CreateSessionToken(clientID, sessionID string, validity time.Duration) (token string, actualValidity time.Duration)

	// DecodeSessionToken and return its claims
	DecodeSessionToken(sessionToken string, signedNonce string, nonce string) (
		clientID string, sessionID string, err error)

	// GetAlg returns the supported security format and authentication algorithm.
	// This uses the vocabulary as defined in the TD.
	// JWT: "ES256", "ES512", "EdDSA"
	// paseto: "local" (symmetric), "public" (asymmetric)
	GetAlg() (string, string)

	// Login with a password and obtain a new session token with limited duration
	// This creates a new session that remains valid until logout or expiry.
	// The token must be refreshed to keep the session alive.
	//
	// This returns the token and the validity period in seconds before it must be refreshed.
	// If the login fails this returns an error
	Login(login string, password string) (token string, validity time.Duration, err error)

	// Logout removes the session and invalidates the all tokens of this client
	Logout(clientID string)

	// RefreshToken issues a new session token with an updated expiry time.
	// This extends the life of the session.
	//
	//	clientID Client whose token to refresh
	//	oldToken must be valid
	//
	// This returns the token and the validity period before it must be refreshed,
	// If the clientID is unknown or oldToken is no longer valid this returns an error
	RefreshToken(clientID string, oldToken string) (newToken string, validity time.Duration, err error)

	// Set the URI where to login
	SetAuthServerURI(authServiceURI string)

	// ValidatePassword checks if the given password is valid for the client
	ValidatePassword(clientID string, password string) (err error)

	// ValidateToken validates the auth token and returns the token clientID.
	// If the token is invalid an error is returned
	ValidateToken(token string) (clientID string, sessionID string, err error)
}

package transports

import (
	"time"

	"github.com/hiveot/hivekit/go/wot/td"
)

// Interface of the authentication capability for setting TD security scheme
// and authenticating incoming connections.
type IAuthenticator interface {

	// AddSecurityScheme adds the wot securityscheme to the given TD
	AddSecurityScheme(tdoc *td.TD)

	// DecodeToken decodes the given token using the configured authenticator.
	// DecodeToken(token string, signedNonce string, nonce string) (
	// clientID string, issuedAt time.Time, validUntil time.Time, err error)

	// GetAlg returns the supported security format and authentication algorithm.
	// This uses the vocabulary as defined in the TD.
	// JWT: "ES256", "ES512", "EdDSA"
	// paseto: "local" (symmetric), "public" (asymmetric)
	// GetAlg() (string, string)

	// ValidatePassword checks if the given password is valid for the client
	// ValidatePassword(clientID string, password string) (err error)

	// ValidateToken verifies the token and client are valid.
	// This returns an error if the token is invalid, the token has expired,
	// or the client is not a valid and enabled client.
	ValidateToken(token string) (clientID string, validUntil time.Time, err error)
}

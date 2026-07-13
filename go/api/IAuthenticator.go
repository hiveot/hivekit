package api

import (
	"time"

	"github.com/hiveot/hivekit/go/api/td"
)

// const IAuthenticatorModuleType = "IAuthenticator"

// Interface of the authentication capability for setting TD security scheme
// and authenticating incoming connections.

type IAuthenticator interface {

	// AddSecurityScheme adds the wot securityscheme that describes this authenticator to the given TD
	AddSecurityScheme(tdoc *td.TD)

	// ValidateDigest checks if the given password digest is valid for the client
	// ValidateDigest(clientID string, digest string) (err error)

	// ValidatePassword checks if the given password is valid for the client
	// ValidatePassword(clientID string, password string) (err error)

	// ValidateClient verifies the secret is valid for the claimed clientID.
	//
	// This returns the validated clientID and the time the secret was issued and is valid for.
	// This returns an error if the secret is invalid, has expired, or the client is blocked..
	ValidateClient(claimedClientID string, secret string) (clientID string, issuedAt time.Time, validUntil time.Time, err error)
}

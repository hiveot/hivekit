package api

import (
	"time"

	"github.com/hiveot/hivekit/go/api/td"
)

// Interface of the authentication capability for setting TD security scheme
// and authenticating incoming connections.

type IAuthenticator interface {

	// AddSecurityScheme adds the wot securityscheme that describes this authenticator to the given TD
	AddSecurityScheme(tdoc *td.TD)

	// Return the role of the client
	GetRole(clientID string) string

	// ValidateClient verifies the secret is valid for the claimed clientID.
	//
	// This returns the validated clientID and the time the secret was issued and is valid for.
	// This returns an error if the secret is invalid, has expired, or the client is blocked..
	ValidateClient(claimedClientID string, secret string) (clientID string, issuedAt time.Time, validUntil time.Time, err error)
}

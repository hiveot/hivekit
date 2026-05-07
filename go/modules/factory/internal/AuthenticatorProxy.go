package internal

import (
	"fmt"
	"time"

	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules/transports"
)

type AuthenticatorProxy struct {
	impl   transports.IAuthenticator
	noAuth bool
}

// Set the TD security scheme for the authentication
// If no authenticator is set then
func (ap *AuthenticatorProxy) AddSecurityScheme(tdoc *td.TD) {
	// proxy doesnt do anything unless implementation is set
	if ap.impl != nil {
		ap.impl.AddSecurityScheme(tdoc)
	} else if ap.noAuth {
		tdoc.AddSecurityScheme(td.SecSchemeNoSec, td.SecurityScheme{
			Name: td.SecSchemeNoSec,
		})
	}
}

// Set the authenticator implementation
// If nil is provided then disable authentication
func (ap *AuthenticatorProxy) SetAuthenticator(actual transports.IAuthenticator) {
	ap.impl = actual
	ap.noAuth = actual == nil
}

// Validate the token
// This passes it on to the authenticator set with SetAuthenticator.
// If no authenticator is setup and noauth is true then always accept the token.
func (ap *AuthenticatorProxy) ValidateToken(token string) (
	clientID string, issuedAt time.Time, validUntil time.Time, err error) {

	if ap.impl != nil {
		return ap.impl.ValidateToken(token)
	}
	if ap.noAuth {
		return "", issuedAt, validUntil, nil
	}
	return "", issuedAt, validUntil, fmt.Errorf("No authenticator has been configured")
}

func NewAuthenticatorProxy() *AuthenticatorProxy {
	ap := &AuthenticatorProxy{
		noAuth: false,
	}
	return ap
}

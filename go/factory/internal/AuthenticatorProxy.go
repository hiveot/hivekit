package internal

import (
	"fmt"
	"time"

	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules/transports"
)

type AuthenticatorProxy struct {
	impl transports.IAuthenticator
}

func (ap *AuthenticatorProxy) AddSecurityScheme(tdoc *td.TD) {
	// proxy doesnt do anything unless implementation is set
	if ap.impl != nil {
		ap.impl.AddSecurityScheme(tdoc)
	}
}

// Set the authenticator implementation
func (ap *AuthenticatorProxy) SetAuthenticator(actual transports.IAuthenticator) {
	ap.impl = actual
}

func (ap *AuthenticatorProxy) ValidateToken(token string) (
	clientID string, issuedAt time.Time, validUntil time.Time, err error) {

	if ap.impl != nil {
		return ap.impl.ValidateToken(token)
	}
	return "", issuedAt, validUntil, fmt.Errorf("No authenticator has been configured")
}

func NewAuthenticatorProxy() *AuthenticatorProxy {
	ap := &AuthenticatorProxy{}
	return ap
}

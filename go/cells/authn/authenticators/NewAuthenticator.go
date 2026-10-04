package authn_authenticators

import (
	"crypto/ecdsa"
	"crypto/ed25519"

	"github.com/hiveot/hivekit/go/cells/authn"
	"github.com/hiveot/hivekit/go/cells/authn/authenticators/internal"
)

// NewPasetoAuthenticator returns a new instance of a Paseto token authenticator using the given signing key
// the session manager is used
//
//	authnStore is the client storage to validate the client exists
//	signingKey is the key the tokens are signed with
//	authServerURI is the login endpoint URI (optional)
func NewPasetoAuthenticator(
	authnStore authn.IAuthnStore, signingKey ed25519.PrivateKey, authServerURI string) authn.IAuthnAuthenticator {
	pasetoAuthn := internal.NewPasetoAuthenticatorImpl(authnStore, signingKey, authServerURI)
	return pasetoAuthn
}

// NewJWTAuthenticator returns a new instance of a JWT token authenticator using the
// given signing key.
//
// The token includes the client's role and login URI.
//
//	authnStore is the client storage to validate the client exists and its role
//	signingKey is the key the tokens are signed with
//	authServerURI is the login endpoint URI (optional)
func NewJWTAuthenticator(
	authnStore authn.IAuthnStore, signingKey *ecdsa.PrivateKey, authServerURI string) authn.IAuthnAuthenticator {
	pasetoAuthn := internal.NewJWTAuthenticatorImpl(authnStore, signingKey, authServerURI)
	return pasetoAuthn
}

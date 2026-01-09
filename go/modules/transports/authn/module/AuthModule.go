package module

import (
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/modules/transports/authn"
	"github.com/hiveot/hivekit/go/modules/transports/httptransport"
)

// AuthnModule is a module that issues authentication tokens.
// This registers authn endpoints on the given http server
// This implements IHiveModule and IAuthnModule interfaces.
type AuthnModule struct {
	transports.TransportModuleBase

	// The http/tls server to register endpoints
	httpServer httptransport.IHttpServer

	// The primary authenticator
	authenticator transports.IAuthenticator
}

// Return the authenticator for use by other modules
func (m *AuthnModule) GetAuthenticator() transports.IAuthenticator {
	return m.authenticator
}

// GetAuthServerURI returns the URI of the authentication server to include in the TD security scheme
// FIXME: Should this be some kind of authorization flow with a web page?
// This is currently just the login endpoint (post /authn/login).
// The http server might need to include a web page where users can enter their login
// name and password, although that won't work for machines... tbd
//
// Note that web browsers do not directly access the runtime endpoints.
// Instead a web server (hiveoview or other) provides the user interface.
// Including the auth endpoint here is currently just a hint. How to integrate this?
func (m *AuthnModule) GetAuthServerURI() string {
	return authn.HttpPostLoginPath
}

func NewAuthnModule() *AuthnModule {
	m := AuthnModule{}
	return &m
}

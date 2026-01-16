package module

import (
	"path"
	"time"

	"github.com/hiveot/hivekit/go/modules/authn"
	"github.com/hiveot/hivekit/go/modules/transports"
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
	baseURL := m.httpServer.GetConnectURL()
	loginURL := path.Join(baseURL, authn.HttpPostLoginPath)
	return loginURL
}

// Login verifies the password and generates a new limited authentication token
//
// This uses the configured session authenticator.
func (m *AuthnModule) Login(clientID string, password string) (
	newToken string, validity time.Duration, err error) {

	// the module uses the configured authenticator
	newToken, validity, err = m.authenticator.Login(clientID, password)
	_ = validity
	return newToken, validity, err
}

// Logout disables the client's sessions
//
// This uses the configured session authenticator.
func (m *AuthnModule) Logout(clientID string) {

	// the module uses the configured authenticator
	m.authenticator.Logout(clientID)
}

// RefreshToken refreshes the auth token using the session authenticator.
//
// This uses the configured session authenticator.
func (m *AuthnModule) RefreshToken(clientID, oldToken string) (
	newToken string, validity time.Duration, err error) {

	newToken, validity, err = m.authenticator.RefreshToken(clientID, oldToken)
	return newToken, validity, err
}

// Start the authentication module and listen for login and token refresh requests
func (m *AuthnModule) Start() error {
	err := m.createRoutes()
	return err
}

func (m *AuthnModule) Stop() {
}

func NewAuthnModule(httpServer httptransport.IHttpServer, authenticator transports.IAuthenticator) *AuthnModule {
	m := AuthnModule{
		httpServer:    httpServer,
		authenticator: authenticator,
	}
	return &m
}

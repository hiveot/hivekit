package authn

import (
	"time"

	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/transports"
)

const DefaultAuthnThingID = "authn"

const (
	// HttpPostLoginPath is the fixed authentication endpoint of the hub
	HttpPostLoginPath   = "/authn/login"
	HttpPostLogoutPath  = "/authn/logout"
	HttpPostRefreshPath = "/authn/refresh"
)

// helper for building a login request message
// tbd this should probably go elsewhere.
type UserLoginArgs struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// Authenticate server for login and refresh tokens
// This implements the IAuthenticator API for use by other services
// Use authapi.AuthClient to create a client for logging in.
type IAuthnModule interface {
	modules.IHiveModule
	transports.IAuthenticator

	// Return the authenticator for use by other modules
	GetAuthenticator() transports.IAuthenticator

	// Login verifies the password and generates a new limited authentication token
	Login(clientID string, password string) (
		newToken string, validity time.Duration, err error)

	// Logout disables the client's sessions
	Logout(clientID string)

	// RefreshToken refreshes the auth token using the session authenticator.
	// This uses the configured session authenticator.
	RefreshToken(clientID, oldToken string) (
		newToken string, validity time.Duration, err error)
}

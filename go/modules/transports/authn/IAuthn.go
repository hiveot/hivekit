package authn

import (
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

// Authenticate server for login and refresh tokens
// This implements the IAuthenticator API for use by other services
// Use authapi.AuthClient to create a client for logging in.
type IAuthnModule interface {
	modules.IHiveModule
	transports.IAuthenticator

	// Return the authenticator for use by other modules
	GetAuthenticator() transports.IAuthenticator
}

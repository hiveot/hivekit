// package authnapi with messaging definitions for the authn user service
package authn

// AuthnUserThingID is the Thing instance ID of the user facing auth service.
const AuthnUserServiceID = "authn:user"

// RRN Thing property, event and action affordance names
const (
	// Property names

	// Event names

	// Action names
	UserActionGetProfile    = "getProfile"
	UserActionLogout        = "Logout"
	UserActionRefreshToken  = "refreshToken"
	UserActionSetPassword   = "setPassword"
	UserActionUpdateProfile = "updateProfile"
)

// well-known HTTP API endpoints - these must match the TD
const (
	// HttpPostLoginPath is the http authentication endpoint of the module
	HttpPostLoginPath   = "/authn/login"
	HttpPostLogoutPath  = "/authn/logout"
	HttpPostRefreshPath = "/authn/refresh"
	HttpGetProfilePath  = "/authn/profile"
)

// Definition of the login request message
// used in http and rrn messaging
type UserLoginArgs struct {
	UserName string `json:"username"`
	Password string `json:"password"`
}

// UserSetPasswordArgs defines the arguments of the setClientPassword function
// Set Client Password - Update the password of a consumer
//
// Client ID and password
type UserSetPasswordArgs struct {

	// ClientID with Client ID
	ClientID string `json:"clientID,omitempty"`

	// Password with Password
	Password string `json:"password,omitempty"`
}

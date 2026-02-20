package authn

import (
	"time"

	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/wot/td"
)

// This module exposes two services, one admin service and one user oriented service
const AdminServiceID = "authnAdmin"
const UserServiceID = "authnUser"

// Predefined roles of a client
// The roles are hierarchical in permissions:
// Authorization using these roles is applied through an authz service.
// Custom roles can be added if needed but their persmissions need to be
// managed in the authz service.
const (

	// ClientRoleNone means that the client has no permissions.
	// It can not do anything until the role is upgraded to viewer or better
	ClientRoleNone string = "none"

	// ClientRoleViewer for users that can view information for devices/services
	// they have access to.
	// Viewers cannot invoke actions or change configuration.
	ClientRoleViewer string = "viewer"

	// ClientRoleAgent for devices and services.
	//
	// Agents publish device information for the devices/services it manages and
	// receive request for those devices/services.
	ClientRoleAgent string = "agent"

	// ClientRoleOperator for users that operate devices and services.
	//
	// Operators can view and control devices/services they have access to but
	// not configure them.
	ClientRoleOperator string = "operator"

	// ClientRoleManager for users that manage devices.
	//
	// Managers can view, control and configure devices/services they have access to.
	ClientRoleManager string = "manager"

	// ClientRoleAdmin for users that administer the system.
	//
	// Administrators can view, control and configure all devices and services.
	ClientRoleAdmin string = "admin"

	// ClientRoleService for Service role
	//
	// Services are equivalent to an admin user and agent for devices/services they
	// have access to.
	ClientRoleService string = "service"
)

// ClientProfile defines a Client Profile data schema.
//
// This contains client information of device agents, services and consumers
type ClientProfile struct {

	// ClientID with the unique client ID
	ClientID string `json:"clientID,omitempty"`

	// Disabled flag to enable/disable the client account
	Disabled bool `json:"disabled,omitempty"`

	// DisplayName of the client
	DisplayName string `json:"displayName,omitempty"`

	// PubKey with public key in PEM format intended for encryption
	PubKeyPem string `json:"pubKey,omitempty"`

	// Role of the client when the account is enabled
	// note that roles can only be updated using UpdateProfile by administrators.
	Role string `json:"role,omitempty"`

	// TimeCreated time the client account was created
	TimeCreated string `json:"created,omitempty"`

	// TimeUpdated time the client was last updated
	TimeUpdated string `json:"updated,omitempty"`
}

// Authentication server for login and refresh tokens.
// This implements the facilities for managing clients.
type IAuthnModule interface {
	modules.IHiveModule

	// AddClient add a new client account. This fails if the client already exists.
	// Use authenticator's SetPassword or CreateToken to obtain a token to connect.
	AddClient(clientID string, displayName string, role string) error

	// AddSecurityScheme adds the wot securityscheme to the given TD
	AddSecurityScheme(tdoc *td.TD)

	// DecodeToken decodes the given token using the configured authenticator.
	DecodeToken(token string, signedNonce string, nonce string) (
		clientID string, issuedAt time.Time, validUntil time.Time, err error)

	// Return the client authenticator for use by transport modules
	// GetAuthenticator() transports.IAuthenticator

	// GetAlg returns the supported security format and authentication algorithm.
	// This uses the vocabulary as defined in the TD.
	// JWT: "ES256", "ES512", "EdDSA"
	// paseto: "local" (symmetric), "public" (asymmetric)
	// GetAlg() (string, string)

	// GetProfile Get the client profile
	GetProfile(clientID string) (profile ClientProfile, err error)

	// GetProfiles Get Profiles
	// Get a list of all client profiles
	GetProfiles() (profiles []ClientProfile, err error)

	// Login with a password and obtain a new authentication token with limited duration.
	// The token must be refreshed before it expires.
	//
	// Token validation is determined through configuration.
	//
	// This returns the authentication token and the expiration time before it must be refreshed.
	// If the login fails this returns an error
	Login(login string, password string) (token string, validUntil time.Time, err error)

	// Logout invalidates all tokens of this client issued before now.
	Logout(clientID string)

	// RefreshToken issues a new authentication token with an updated expiry time.
	// This extends the life of the session.
	//
	//	clientID Client whose token to refresh
	//	oldToken must be valid
	//
	// This returns the token and the validity time before it must be refreshed,
	// If the clientID is unknown or oldToken is no longer valid this returns an error
	RefreshToken(clientID string, oldToken string) (newToken string, validUntil time.Time, err error)

	// RemoveClient removes client account
	RemoveClient(clientID string) error

	// SetPassword sets a client's password for use with Login()
	SetPassword(clientID string, password string) error

	// SetRole sets a client's role.
	// Like passwords only an admin or service can update roles.
	SetRole(clientID string, role string) error

	// UpdateProfile changes a client's profile.
	// Only administrators can update the role. (senderID has role admin or service)
	UpdateProfile(senderID string, profile ClientProfile) error

	// ValidatePassword checks if the given password is valid for the client
	ValidatePassword(clientID string, password string) (err error)

	// ValidateToken verifies the token and client are valid.
	// This returns an error if the token is invalid, the token has expired,
	// or the client is not a valid and enabled client.
	ValidateToken(token string) (clientID string, validUntil time.Time, err error)
}

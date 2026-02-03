package authn

import (
	"github.com/hiveot/hivekit/go/modules"
)

// This module exposes two services, one admin service and one user oriented service
const AdminServiceID = "authnAdmin"
const UserServiceID = "authnUser"

// ClientRole enumerator
//
// Identifies the client's role
type ClientRole string

// Predefined roles of a client
// The roles are hierarchical in permissions:
// Authorization using these roles is applied through an authz service.
// Custom roles can be added if needed but their persmissions need to be
// managed in the authz service.
const (

	// ClientRoleNone means that the client has no permissions.
	// It can not do anything until the role is upgraded to viewer or better
	ClientRoleNone ClientRole = "none"

	// ClientRoleViewer for users that can view information for devices/services
	// they have access to.
	// Viewers cannot invoke actions or change configuration.
	ClientRoleViewer ClientRole = "viewer"

	// ClientRoleAgent for devices and services.
	//
	// Agents publish device information for the devices/services it manages and
	// receive request for those devices/services.
	ClientRoleAgent ClientRole = "agent"

	// ClientRoleOperator for users that operate devices and services.
	//
	// Operators can view and control devices/services they have access to but
	// not configure them.
	ClientRoleOperator ClientRole = "operator"

	// ClientRoleManager for users that manage devices.
	//
	// Managers can view, control and configure devices/services they have access to.
	ClientRoleManager ClientRole = "manager"

	// ClientRoleAdmin for users that administer the system.
	//
	// Administrators can view, control and configure all devices and services.
	ClientRoleAdmin ClientRole = "admin"

	// ClientRoleService for Service role
	//
	// Services are equivalent to an admin user and agent for devices/services they
	// have access to.
	ClientRoleService ClientRole = "service"
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
	// note that roles are not updated in UpdateProfile.
	Role ClientRole `json:"role,omitempty"`

	// TimeCreated time the client account was created
	TimeCreated string `json:"created,omitempty"`

	// TimeUpdated time the client was last updated
	TimeUpdated string `json:"updated,omitempty"`
}

// Authenticate server for login and refresh tokens.
// This implements the facilities for managing clients.
type IAuthnModule interface {
	modules.IHiveModule

	// AddClient add a new client account. This fails if the client already exists.
	// Use authenticator's SetPassword or CreateToken to obtain a token to connect.
	AddClient(clientID string, displayName string, role ClientRole, pubKey string) error

	// Return the client authenticator for use by transport modules
	GetAuthenticator() IAuthenticator

	// GetProfile Get the client profile
	GetProfile(clientID string) (profile ClientProfile, err error)

	// GetProfiles Get Profiles
	// Get a list of all client profiles
	GetProfiles() (profiles []ClientProfile, err error)

	// RemoveClient removes client account
	RemoveClient(clientID string) error

	// SetPassword sets a client's password for use with Login()
	SetPassword(clientID string, password string) error

	// SetRole sets a client's role.
	// Like passwords only an admin or service can update roles.
	SetRole(clientID string, role ClientRole) error

	// UpdateProfile changes a client's profile.
	// Only administrators can update the role. (senderID has role admin or service)
	UpdateProfile(senderID string, profile ClientProfile) error
}

package authn

import (
	_ "embed"

	"github.com/hiveot/hivekit/go/api"
)

// admin tokens last longer
const AdminTokenValidityDays = 365

// Session token validity for client types
const (
	DefaultConsumerTokenValidityDays = 30
	DefaultDeviceTokenValidityDays   = 90
	DefaultServiceTokenValidityDays  = 365
)

// Embed admin service TD
//
//go:embed "authn-admin-td.json"
var AuthnAdminTD []byte

// Embed user service TM
//
//go:embed "authn-user-td.json"
var AuthnUserTD []byte

// This service exposes two services in this cell, one for administrator use, one for consumer use
// and one for user storage.
// The cell type is also the instance ID as these are singletons
const (
	AuthnServiceCellType     = "authn"
	AuthnAdminServiceThingID = "authn:admin"
	AuthnUserServiceThingID  = "authn:user"
)

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

	// ClientRoleDevice for devices.
	//
	// Devices publish Thing information for itself and possible nested devices,
	// and handle request for those devices.
	ClientRoleDevice string = "device"

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
	// Services are equivalent to both an admin user and exposed things.
	ClientRoleService string = "service"
)

// ClientProfile defines a Client Profile data schema.
//
// This contains client information of devices, services and consumers
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

// Interface of the authentication service
// This supports both the admin and user management services
type IAuthnService interface {
	api.IHiveCell
	GetAdminService() IAuthnAdminService
	GetUserService() IAuthnUserService
}

// Interface of the authentication admin service for managing clients and provide
// the session manager and authenticator.
type IAuthnAdminService interface {

	// AddClient add a new client account. This fails if the client already exists.
	//
	// Use session authenticator's SetPassword or CreateToken to obtain a session
	// token to connect with.
	//
	//	clientID is the account ID of the client
	//	displayName is the friendly name of the client
	//	role is the client role, eg ClientRoleViewer, ... ClientRoleDevice
	AddClient(clientID string, displayName string, role string) error

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
	SetRole(clientID string, role string) error

	// UpdateProfile changes a client's profile.
	// Only administrators can update the role. (senderID has role admin or service)
	UpdateProfile(senderID string, profile ClientProfile) error
}

// Interface of the user self-management service for login, logout and edit profile.
type IAuthnUserService interface {

	// GetProfile Get the sender's profile
	GetProfile(senderID string) (profile ClientProfile, err error)

	// obtain the session manager for authentication use by users
	GetSessionManager() ISessionManager

	// SetPassword sets a sender's password for use with Login()
	SetPassword(senderID string, password string) error

	// UpdateProfile changes a sender's profile.
	// The profile role cannot be updated.
	UpdateProfile(senderID string, profile ClientProfile) error
}

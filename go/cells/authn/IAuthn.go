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
	DefaultAuthnAdminThingID = "authn:admin"
	DefaultAuthnUserThingID  = "authn:user"
	AuthnAdminServiceType    = "authn:admin"
	AuthnUserServiceType     = "authn:user"
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

	// TimeCreated ISO time the client account was created
	TimeCreated string `json:"created,omitempty"`

	// TimeUpdated ISO time the client was last updated
	TimeUpdated string `json:"updated,omitempty"`
}

// Interface of the authentication service
// This supports both the admin and user management services
type IAuthnService interface {
	api.IHiveCell
	GetAdminService() IAuthnAdminService
	GetUserService() IAuthnUserService
	GetSessionManager() ISessionManager
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

	// SetPassword sets a sender's password for use with Login()
	SetPassword(senderID string, password string) error

	// UpdateProfile changes a sender's profile.
	// The profile role cannot be updated.
	UpdateProfile(senderID string, profile ClientProfile) error
}

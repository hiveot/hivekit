package authn

import (
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/transports"
)

// This module exposes two services, one admin service and one user oriented service
const AdminServiceID = "authnAdmin"
const UserServiceID = "authnUser"

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

// Authenticate server for login and refresh tokens.
// This implements the facilities for managing clients.
type IAuthnModule interface {
	modules.IHiveModule

	// AddClient add a new client account. This fails if the client already exists.
	// Use authenticator's SetPassword or CreateToken to obtain a token to connect.
	AddClient(clientID string, displayName string, role string, pubKey string) error

	// Return the client authenticator for use by transport modules
	GetAuthenticator() transports.IAuthenticator

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

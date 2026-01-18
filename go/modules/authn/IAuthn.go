package authn

import (
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/transports"
)

const DefaultAuthnThingID = "authn"

const (
	// HttpPostLoginPath is the http authentication endpoint of the module
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

// ClientType enumerator
//
// identifies the client's category
type ClientType string

const (

	// ClientTypeAgent for Agent
	//
	// Agents represent one or more devices
	ClientTypeAgent ClientType = "agent"

	// ClientTypeService for Service
	//
	// Service enrich information
	ClientTypeService ClientType = "service"

	// ClientTypeConsumer for Consumer
	//
	// Consumers are end-users of information
	ClientTypeConsumer ClientType = "consumer"
)

// ClientProfile defines a Client Profile data schema.
//
// This contains client information of device agents, services and consumers
type ClientProfile struct {

	// ClientID with Client ID
	ClientID string `json:"clientID,omitempty"`

	// ClientType with Client Type
	ClientType ClientType `json:"clientType,omitempty"`

	// Disabled
	//
	// This client account has been disabled
	Disabled bool `json:"disabled,omitempty"`

	// DisplayName with
	DisplayName string `json:"displayName,omitempty"`

	// PubKey with Public Key
	PubKey string `json:"pubKey,omitempty"`

	// Updated with Client name or auth updated timestamp
	Updated string `json:"updated,omitempty"`
}

// Authenticate server for login and refresh tokens
// This implements the IAuthenticator API for use by other services
// Use authapi.AuthClient to create a client for logging in.
type IAuthnModule interface {
	modules.IHiveModule
	transports.IAuthenticator

	// Return the authenticator for use by other modules
	GetAuthenticator() transports.IAuthenticator

	// GetProfile Get Client Profile
	GetProfile(senderID string) (resp ClientProfile, err error)

	// UpdateName Request changing the display name of the current client
	UpdateName(senderID string, newName string) error

	// UpdatePassword Update Password
	// Request changing the password of the current client
	UpdatePassword(senderID string, password string) error

	// UpdatePubKey Update Public Key
	// Request changing the public key on file of the current client.
	UpdatePubKey(senderID string, publicKeyPEM string) error
}

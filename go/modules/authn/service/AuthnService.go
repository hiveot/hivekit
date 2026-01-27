package authnservice

import (
	"crypto/ed25519"
	"fmt"
	"path"
	"time"

	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/authn"
	"github.com/hiveot/hivekit/go/modules/authn/authenticators"
	"github.com/hiveot/hivekit/go/modules/authn/service/authnstore"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/msg"
	"github.com/hiveot/hivekit/go/utils"
)

// AuthnService is a module that manages clients and issues authentication tokens.
//
// This implements IHiveModule and IAuthnModule interfaces and is facade for the
// account store and authenticator.
type AuthnService struct {
	modules.HiveModuleBase

	config authn.AuthnConfig

	// The http/tls server to register endpoints
	httpServer transports.IHttpServer

	// The primary authenticator
	authenticator transports.IAuthenticator
	//
	authnStore authnstore.IAuthnStore

	// Messaging API handlers
	userHttpHandler *UserHttpHandler
}

func (m *AuthnService) AddClient(
	clientID string, displayName string, role authn.ClientRole, pubKey string) error {

	_, err := m.authnStore.GetProfile(clientID)
	if err == nil {
		return fmt.Errorf("Account for client '%s' already exists", clientID)
	}

	newProfile := authn.ClientProfile{
		ClientID:    clientID,
		DisplayName: displayName,
		Role:        role,
		PubKey:      pubKey,
	}
	return m.authnStore.Add(newProfile)
}

// Return the authenticator for use by other modules
func (m *AuthnService) GetAuthenticator() transports.IAuthenticator {
	return m.authenticator
}

// GetConnectURL returns the URI of the authentication server to include in the TD
// security scheme.
//
// This is currently just the base for the login endpoint (post {base}/authn/login).
// The http server might need to include a web page where users can enter their login
// name and password, although that won't work for machines... tbd
//
// Note that web browsers do not directly access the runtime endpoints.
// Instead a web server (hiveoview or other) provides the user interface.
// Including the auth endpoint here is currently just a hint. How to integrate this?
func (m *AuthnService) GetConnectURL() string {
	baseURL := m.httpServer.GetConnectURL()
	loginURL := path.Join(baseURL, HttpPostLoginPath)
	return loginURL
}

// GetProfile return the client's profile
func (m *AuthnService) GetProfile(clientID string) (profile authn.ClientProfile, err error) {
	return m.authnStore.GetProfile(clientID)
}

// GetProfile return a list of client profiles
func (m *AuthnService) GetProfiles() (profiles []authn.ClientProfile, err error) {
	return m.authnStore.GetProfiles()
}

// Handle requests to be served by this module
func (m *AuthnService) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) error {

	//TODO: how to handle read property requests? admin or user?
	if req.ThingID == AuthnAdminServiceID {
		return HandleAuthnAdminRequest(m, req, replyTo)
	} else if req.ThingID == AuthnUserServiceID {
		return HandleAuthnUserRequest(m, req, replyTo)
	} else {
		// forward
		return m.HiveModuleBase.HandleRequest(req, replyTo)
	}

}

// Login verifies the password and generates a new limited authentication token
//
// This uses the configured session authenticator.
func (m *AuthnService) Login(clientID string, password string) (
	newToken string, validUntil time.Time, err error) {

	// the module uses the configured authenticator
	newToken, validUntil, err = m.authenticator.Login(clientID, password)
	return newToken, validUntil, err
}

// Logout disables the client's sessions
//
// This uses the configured session authenticator.
func (m *AuthnService) Logout(clientID string) {

	// the module uses the configured authenticator
	m.authenticator.Logout(clientID)
}

// RefreshToken refreshes the auth token using the session authenticator.
//
// This uses the configured session authenticator.
func (m *AuthnService) RefreshToken(clientID, oldToken string) (
	newToken string, validUntil time.Time, err error) {

	newToken, validUntil, err = m.authenticator.RefreshToken(clientID, oldToken)
	return newToken, validUntil, err
}

// Remove a client
func (m *AuthnService) RemoveClient(clientID string) error {
	return m.authnStore.Remove(clientID)
}

// Change the role of a client
func (m *AuthnService) SetRole(clientID string, role string) error {
	return m.authnStore.SetRole(clientID, role)
}

// Change the password of a client
func (m *AuthnService) SetPassword(clientID string, password string) error {
	return m.authenticator.SetPassword(clientID, password)
}

// Start the authentication module and listen for login and token refresh requests
// This reloads the signing key, opens the password store and starts the
// authenticator instance.
//
// yamlConfig with module startup configuration (todo)
func (m *AuthnService) Start(yamlConfig string) (err error) {

	if m.httpServer != nil {
		m.userHttpHandler = NewUserHttpHandler(m.authenticator, m.httpServer)
	}
	passwordFile := m.config.PasswordFile
	encryption := m.config.Encryption

	authnStore := authnstore.NewAuthnFileStore(passwordFile, encryption)

	clientID := "authn"
	signingPrivKey, _, err := utils.LoadCreateKeyPair(
		clientID, m.config.KeysDir, utils.KeyTypeED25519)
	if err != nil {
		return err
	}

	m.authenticator = authenticators.NewPasetoAuthenticator(
		authnStore, signingPrivKey.(ed25519.PrivateKey))

	return err
}

func (m *AuthnService) Stop() {
}

// UpdateProfile update the client profile
// only administrators are allowed to update the role
func (m *AuthnService) UpdateProfile(senderID string, newProfile authn.ClientProfile) error {
	senderProf, err := m.authnStore.GetProfile(senderID)
	if err != nil {
		return fmt.Errorf("Unknown sender '%s'", senderID)
	}
	clientProf, err := m.authnStore.GetProfile(newProfile.ClientID)
	if err != nil {
		return fmt.Errorf("Unknown client '%s'", newProfile.ClientID)
	}
	if senderID != newProfile.ClientID {
		// only admin roles can update client profiles
		if senderProf.Role != authn.ClientRoleAdmin && senderProf.Role != authn.ClientRoleService {
			return fmt.Errorf("Sender '%s' is not admin, not allowed to update profile", senderID)
		}
	} else {
		// client cannot change its own role
		if newProfile.Role != "" && newProfile.Role != clientProf.Role {
			return fmt.Errorf("Client '%s' is not allowed to change its role", senderID)
		}
	}
	return m.authnStore.UpdateProfile(newProfile)
}

// Create a new authentication module.
//
// authnConfig contains the password storage and token management configuration
// httpServer is optional and used to make http endpoints available for login, logout and token refresh.
func NewAuthnService(authnConfig authn.AuthnConfig, httpServer transports.IHttpServer) *AuthnService {

	m := &AuthnService{
		config:     authnConfig,
		httpServer: httpServer,
	}
	var _ modules.IHiveModule = m // interface check
	var _ authn.IAuthnModule = m  // interface check
	return m
}

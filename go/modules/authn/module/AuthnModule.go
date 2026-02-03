package module

import (
	"crypto/ed25519"
	"fmt"
	"net/url"

	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/authn"
	"github.com/hiveot/hivekit/go/modules/authn/module/authenticators"
	"github.com/hiveot/hivekit/go/modules/authn/module/authnstore"
	"github.com/hiveot/hivekit/go/modules/authn/server"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/msg"
	"github.com/hiveot/hivekit/go/utils"
)

// AuthnModule is a module that manages clients and issues authentication tokens.
//
// This implements IHiveModule and IAuthnModule interfaces and is facade for the
// account store and authenticator.
type AuthnModule struct {
	modules.HiveModuleBase

	config authn.AuthnConfig

	// The http/tls server to register endpoints
	httpServer transports.IHttpServer

	// The primary authenticator
	authenticator authn.IAuthenticator
	//
	authnStore authnstore.IAuthnStore

	// Messaging API handlers
	userHttpHandler *server.UserHttpHandler
}

// Return the authenticator for use by other modules
func (m *AuthnModule) GetAuthenticator() authn.IAuthenticator {
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
func (m *AuthnModule) GetConnectURL() string {
	baseURL := m.httpServer.GetConnectURL()
	loginURL, _ := url.JoinPath(baseURL, server.HttpPostLoginPath)
	return loginURL
}

// GetProfile return the client's profile
func (m *AuthnModule) GetProfile(clientID string) (profile authn.ClientProfile, err error) {
	return m.authnStore.GetProfile(clientID)
}

// GetProfile return a list of client profiles
func (m *AuthnModule) GetProfiles() (profiles []authn.ClientProfile, err error) {
	return m.authnStore.GetProfiles()
}

// Handle requests to be served by this module
func (m *AuthnModule) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) error {

	//TODO: how to handle read property requests? admin or user?
	if req.ThingID == server.AuthnAdminServiceID {
		return server.HandleAuthnAdminRequest(m, req, replyTo)
	} else if req.ThingID == server.AuthnUserServiceID {
		return server.HandleAuthnUserRequest(m, req, replyTo)
	} else {
		// forward
		return m.HiveModuleBase.HandleRequest(req, replyTo)
	}
}

// Remove a client
func (m *AuthnModule) RemoveClient(clientID string) error {
	return m.authnStore.Remove(clientID)
}

// Set the http server to open up the http endpoints
// If an http server is already set then this panics.
func (m *AuthnModule) SetHttpServer(httpServer transports.IHttpServer) {
	if m.httpServer != nil {
		panic("An HTTP server is already set")
	}
	m.userHttpHandler = server.NewUserHttpHandler(m.authenticator, m.httpServer)
}

// Change the password of a client
func (m *AuthnModule) SetPassword(clientID string, password string) error {
	return m.authenticator.SetPassword(clientID, password)
}

// Change the role of a client
func (m *AuthnModule) SetRole(clientID string, role authn.ClientRole) error {
	return m.authnStore.SetRole(clientID, role)
}

// Start the authentication module and listen for login and token refresh requests
// This reloads the signing key, opens the password store and starts the
// authenticator instance.
//
// yamlConfig with module startup configuration (todo)
func (m *AuthnModule) Start(yamlConfig string) (err error) {

	passwordFile := m.config.PasswordFile
	encryption := m.config.Encryption

	m.authnStore = authnstore.NewAuthnFileStore(passwordFile, encryption)

	clientID := "authn"
	signingPrivKey, _, err := utils.LoadCreateKeyPair(
		clientID, m.config.KeysDir, utils.KeyTypeED25519)
	if err != nil {
		return err
	}

	m.authenticator = authenticators.NewPasetoAuthenticator(
		m.authnStore, signingPrivKey.(ed25519.PrivateKey))

	if m.httpServer != nil {
		m.userHttpHandler = server.NewUserHttpHandler(m.authenticator, m.httpServer)
	}
	return err
}

func (m *AuthnModule) Stop() {
}

// UpdateProfile update the client profile
// only administrators are allowed to update the role
func (m *AuthnModule) UpdateProfile(senderID string, newProfile authn.ClientProfile) error {
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

func (svc *AuthnModule) ValidatePassword(clientID, password string) (err error) {
	clientProfile, err := svc.authnStore.VerifyPassword(clientID, password)
	_ = clientProfile
	return err
}

// Create a new authentication module.
//
// Note: to avoid a chicken-and-egg problem between authentication and http server,
// the authentication module can be started without http server. Use SetHttpServer
// to open the http routes.
// The reverse is also supported when using SetAuthValidator on the http server if
// the auth module starts after the http server. In this case protected routes will
// fail until an auth validator is set.
//
// authnConfig contains the password storage and token management configuration
// httpServer to server the http endpoint or nil to not use http.
func NewAuthnModule(authnConfig authn.AuthnConfig, httpServer transports.IHttpServer) *AuthnModule {

	m := &AuthnModule{
		config:     authnConfig,
		httpServer: httpServer,
	}
	var _ modules.IHiveModule = m // interface check
	var _ authn.IAuthnModule = m  // interface check
	return m
}

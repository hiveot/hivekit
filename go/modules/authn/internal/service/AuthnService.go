package service

import (
	"fmt"
	"log/slog"
	"net/url"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/authn"
	authnstore "github.com/hiveot/hivekit/go/modules/authn/internal/store"
	"github.com/hiveot/hivekit/go/modules/transports"
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

	//
	authnStore authnstore.IAuthnStore

	// Messaging API handlers
	userHttpHandler *UserHttpHandler

	// Creation and validation of session tokens
	sessionManager *SessionManager
}

// AddClient adds a client. This fails if the client already exists
// This should only be usable by administrators.
func (m *AuthnService) AddClient(clientID string, displayName string, role string) error {

	_, err := m.authnStore.GetProfile(clientID)
	if err == nil {
		return fmt.Errorf("Account for client '%s' already exists", clientID)
	}

	newProfile := authn.ClientProfile{
		ClientID:    clientID,
		DisplayName: displayName,
		Role:        role,
	}
	return m.authnStore.Add(newProfile)
}

// GetConnectURL returns the URI of the authentication server with protocol to include
// in the TD security scheme.
//
// This is currently just the base for the login endpoint (post {base}/authn/login).
// The http server might need to include a web page where users can enter their login
// name and password, although that won't work for machines... tbd
//
// Note that web browsers do not directly access the runtime endpoints.
// Instead a web server (hiveoview or other) provides the user interface.
// Including the auth endpoint here is currently just a hint. How to integrate this?
func (m *AuthnService) GetConnectURL() (uri string, protocolType string) {
	baseURL := m.httpServer.GetConnectURL()
	loginURL, _ := url.JoinPath(baseURL, HttpPostLoginPath)
	return loginURL, transports.ProtocolTypeWotHttpBasic
}

// GetProfile return the client's profile
func (m *AuthnService) GetProfile(clientID string) (profile authn.ClientProfile, err error) {
	return m.authnStore.GetProfile(clientID)
}

// GetProfile return a list of client profiles
func (m *AuthnService) GetProfiles() (profiles []authn.ClientProfile, err error) {
	return m.authnStore.GetProfiles()
}

func (m *AuthnService) GetSessionManager() authn.ISessionManager {
	return m.sessionManager
}

// Handle requests to be served by this module
func (m *AuthnService) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) error {

	switch req.ThingID {
	case authn.AuthnAdminServiceID:
		return HandleAuthnAdminRequest(m, req, replyTo)
	case authn.AuthnUserServiceID:
		return HandleAuthnUserRequest(m, req, replyTo)
	default:
		// forward
		return m.HiveModuleBase.HandleRequest(req, replyTo)
	}
}

// Remove a client
func (m *AuthnService) RemoveClient(clientID string) error {
	return m.authnStore.Remove(clientID)
}

// Set the http server to open up the http endpoints
// If an http server is already set then this panics.
// func (m *AuthnServer) SetHttpServer(httpServer transports.IHttpServer) {
// 	if m.httpServer != nil {
// 		panic("An HTTP server is already set")
// 	}
// 	m.userHttpHandler = NewUserHttpHandler(m, m.httpServer)
// }

// Change the password of a client
func (m *AuthnService) SetPassword(clientID string, password string) error {
	return m.authnStore.SetPassword(clientID, password)
}

// Change the role of a client
func (m *AuthnService) SetRole(clientID string, role string) error {
	return m.authnStore.SetRole(clientID, role)
}

// Start the authentication module and listen for login and token refresh requests.
//
// Opens the password store and starts the session manager instance.
//
// If an http server is provided this registers the http auth endpoint,
// and set this authn module as the auth validation handler.
//
// yamlConfig with module startup configuration (todo)
func (m *AuthnService) Start() (err error) {

	slog.Info("Start: Starting authn")
	err = m.authnStore.Open()
	if err != nil {
		return err
	}
	err = m.sessionManager.Start()
	if err != nil {
		return err
	}

	// clientID := "authn"
	// signingPrivKey, _, err := utils.LoadCreateKeyPair(
	// 	clientID, m.config.KeysDir, utils.KeyTypeED25519)
	// if err != nil {
	// 	return err
	// }

	// if an http server is provided then register the endpoints
	if m.httpServer != nil {
		// m.httpServer.SetAuthenticator(m)
		m.userHttpHandler = NewUserHttpHandler(m, m.httpServer)
	}
	return err
}

// // RefreshToken requests a new token based on the old token
// // This requires that the existing session is still valid
// func (m *AuthnService) RefreshToken(senderID string, oldToken string) (
// 	newToken string, validUntil time.Time, err error) {

// 	// validation only succeeds if there is an active session
// 	tokenClientID, _, err := m.ValidateToken(oldToken)
// 	if err != nil || senderID != tokenClientID {
// 		return newToken, validUntil, fmt.Errorf("Invalid token or senderID mismatch")
// 	}
// 	// must still be a valid client
// 	prof, err := m.authnStore.GetProfile(senderID)
// 	_ = prof
// 	if err != nil || prof.Disabled {
// 		return newToken, validUntil, fmt.Errorf("Profile for '%s' is disabled", senderID)
// 	}
// 	validityDays := m.config.ConsumerTokenValidityDays
// 	if prof.Role == authnapi.ClientRoleAgent {
// 		validityDays = m.config.AgentTokenValidityDays
// 	} else if prof.Role == authnapi.ClientRoleService {
// 		validityDays = m.config.ServiceTokenValidityDays
// 	}
// 	validity := time.Duration(validityDays) * 24 * time.Hour
// 	newToken, validUntil, err = m.authenticator.CreateToken(senderID, validity)
// 	return newToken, validUntil, err
// }

// Stop closes the client store and releases resources
func (m *AuthnService) Stop() {
	slog.Info("Stop: Stopping authn")
	m.authnStore.Close()
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

// // ValidateToken verifies the token and client are valid.
// func (m *AuthnService) ValidateToken(token string) (
// 	clientID string, validUntil time.Time, err error) {

// 	clientID, issuedAt, validUntil, err := m.authenticator.ValidateToken(token)
// 	if err != nil {
// 		return
// 	}

// 	// check the token is of an active client
// 	// this is set during CreateToken and Login
// 	sessionStart, found := m.sessionStart[clientID]
// 	if !found {
// 		slog.Warn("ValidateToken. No valid session found for client", "clientID", clientID)
// 		return clientID, validUntil, fmt.Errorf("Session is no longer valid")
// 	}
// 	// the session must have started before the token was issued
// 	// this allows a session restart to invalidate all old tokens
// 	if issuedAt.Before(sessionStart) {
// 		slog.Warn("ValidateToken. The token session is no longer valid", "clientID", clientID)
// 		return clientID, validUntil, fmt.Errorf("Session is no longer valid")
// 	}

// 	return clientID, validUntil, err
// }

// Create a new authentication service.
//
// authnConfig contains the password storage and token management configuration
// httpServer to server the http endpoint or nil to not use http.
func NewAuthnService(authnConfig authn.AuthnConfig, httpServer transports.IHttpServer) *AuthnService {

	passwordFile := authnConfig.PasswordFile
	encryption := authnConfig.Encryption
	authnStore := authnstore.NewAuthnFileStore(passwordFile, encryption)
	sessionManager := NewSessionManager(authnStore, authnConfig.KeysDir)

	m := &AuthnService{
		config:         authnConfig,
		httpServer:     httpServer,
		authnStore:     authnStore,
		sessionManager: sessionManager,
		// sessionStart: make(map[string]time.Time),
	}
	var _ modules.IHiveModule = m // interface check
	var _ authn.IAuthnService = m // interface check
	return m
}

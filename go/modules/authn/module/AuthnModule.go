package module

import (
	"crypto/ed25519"
	"fmt"
	"log/slog"
	"net/url"
	"time"

	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/authn"
	"github.com/hiveot/hivekit/go/modules/authn/module/authenticators"
	"github.com/hiveot/hivekit/go/modules/authn/module/authnstore"
	"github.com/hiveot/hivekit/go/modules/authn/server"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/msg"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hivekit/go/wot/td"
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
	authenticator authenticators.IAuthenticator
	//
	authnStore authnstore.IAuthnStore

	// track session start, used in validation
	sessionStart map[string]time.Time

	// Messaging API handlers
	userHttpHandler *server.UserHttpHandler
}

// AddClient adds a client. This fails if the client already exists
// This should only be usable by administrators.
func (m *AuthnModule) AddClient(clientID string, displayName string, role string) error {

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

// AddSecurityScheme adds the authenticator's security scheme to the given TD.
func (m *AuthnModule) AddSecurityScheme(tdoc *td.TD) {
	m.authenticator.AddSecurityScheme(tdoc)
}

// Return the authenticator
// func (m *AuthnModule) GetAuthenticator() authenticators.IAuthenticator {
// 	return m.authenticator
// }

// CreateSessionToken creates a new session token for the client using the configured authenticator.
//
// This creates a session that is valid until logout.
//
//	clientID is the account ID of a known client
//	validity is the token validity period.
//
// This returns the token
func (m *AuthnModule) CreateSessionToken(clientID string, validity time.Duration) (
	token string, validUntil time.Time, err error) {

	//
	createdTime := time.Now()
	m.sessionStart[clientID] = createdTime.Add(-time.Second)

	token, validUntil, err = m.authenticator.CreateToken(clientID, validity)
	return
}

// DecodeToken decodes the given token using the configured authenticator.
// optionally verify the signed nonce using the client's public key.
// This returns the auth info stored in the token.
func (m *AuthnModule) DecodeToken(token string, signedNonce string, nonce string) (
	clientID string, issuedAt time.Time, validUntil time.Time, err error) {
	return m.authenticator.DecodeToken(token, signedNonce, nonce)
}

// GetConnectURL Preturns the URI of the authentication server to include in the TD
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

	//TODO: handle read property requests? admin or user?
	if req.ThingID == server.AuthnAdminServiceID {
		return server.HandleAuthnAdminRequest(m, req, replyTo)
	} else if req.ThingID == server.AuthnUserServiceID {
		return server.HandleAuthnUserRequest(m, req, replyTo)
	} else {
		// forward
		return m.HiveModuleBase.HandleRequest(req, replyTo)
	}
}

// Login with password and generate a session token
// Intended for end-users that want to establish a session.
//
//	clientID is the client to log in
//	password to verify
//
// This returns a session token, its session ID, or an error if failed
func (m *AuthnModule) Login(
	clientID string, password string) (token string, validUntil time.Time, err error) {

	// a user login always creates a session token
	err = m.ValidatePassword(clientID, password)
	if err != nil {
		return "", validUntil, err
	}

	// If a session start time does not exist yet, then record this as the session start.
	sessionStart, found := m.sessionStart[clientID]
	if !found {
		sessionStart = time.Now().Add(-time.Second) // prevent comparison with token iat failing
		m.sessionStart[clientID] = sessionStart
	}

	// create the session to allow token refresh
	validity := time.Hour * time.Duration(24*m.config.ConsumerTokenValidityDays)
	token, validUntil, _ = m.authenticator.CreateToken(clientID, validity)

	return token, validUntil, err
}

// Logout removes the client session
func (m *AuthnModule) Logout(clientID string) {
	_, found := m.sessionStart[clientID]
	if found {
		delete(m.sessionStart, clientID)
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
	m.userHttpHandler = server.NewUserHttpHandler(m, m.httpServer)
}

// Change the password of a client
func (m *AuthnModule) SetPassword(clientID string, password string) error {
	return m.authnStore.SetPassword(clientID, password)
}

// Change the role of a client
func (m *AuthnModule) SetRole(clientID string, role string) error {
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
		m.userHttpHandler = server.NewUserHttpHandler(m, m.httpServer)
	}
	return err
}

// RefreshToken requests a new token based on the old token
// This requires that the existing session is still valid
func (m *AuthnModule) RefreshToken(senderID string, oldToken string) (
	newToken string, validUntil time.Time, err error) {

	// validation only succeeds if there is an active session
	tokenClientID, _, err := m.ValidateToken(oldToken)
	if err != nil || senderID != tokenClientID {
		return newToken, validUntil, fmt.Errorf("Invalid token or senderID mismatch")
	}
	// must still be a valid client
	prof, err := m.authnStore.GetProfile(senderID)
	_ = prof
	if err != nil || prof.Disabled {
		return newToken, validUntil, fmt.Errorf("Profile for '%s' is disabled", senderID)
	}
	validityDays := m.config.ConsumerTokenValidityDays
	if prof.Role == authn.ClientRoleAgent {
		validityDays = m.config.AgentTokenValidityDays
	} else if prof.Role == authn.ClientRoleService {
		validityDays = m.config.ServiceTokenValidityDays
	}
	validity := time.Duration(validityDays) * 24 * time.Hour
	newToken, validUntil, err = m.authenticator.CreateToken(senderID, validity)
	return newToken, validUntil, err
}

// Stop closes the client store and releases resources
func (m *AuthnModule) Stop() {
	m.authnStore.Close()
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

func (m *AuthnModule) ValidatePassword(clientID, password string) (err error) {
	clientProfile, err := m.authnStore.VerifyPassword(clientID, password)
	_ = clientProfile
	return err
}

// ValidateToken verifies the token and client are valid.
func (m *AuthnModule) ValidateToken(token string) (
	clientID string, validUntil time.Time, err error) {

	clientID, issuedAt, validUntil, err := m.authenticator.ValidateToken(token)
	if err != nil {
		return
	}

	// must still be a valid client
	prof, err := m.authnStore.GetProfile(clientID)
	if err != nil || prof.Disabled {
		return clientID, validUntil, fmt.Errorf("Profile for '%s' is disabled", clientID)
	}
	// check the token is of an active client
	// this is set during CreateToken and Login
	sessionStart, found := m.sessionStart[clientID]
	if !found {
		slog.Warn("ValidateToken. No valid session found for client", "clientID", clientID)
		return clientID, validUntil, fmt.Errorf("Session is no longer valid")
	}
	// the session must have started before the token was issued
	// this allows a session restart to invalidate all old tokens
	if issuedAt.Before(sessionStart) {
		slog.Warn("ValidateToken. The token session is no longer valid", "clientID", clientID)
		return clientID, validUntil, fmt.Errorf("Session is no longer valid")
	}

	return clientID, validUntil, err
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
		config:       authnConfig,
		httpServer:   httpServer,
		sessionStart: make(map[string]time.Time),
	}
	var _ modules.IHiveModule = m // interface check
	var _ authn.IAuthnModule = m  // interface check
	return m
}

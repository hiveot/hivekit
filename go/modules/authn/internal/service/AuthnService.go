package service

import (
	"fmt"
	"log/slog"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/authn"
	authnstore "github.com/hiveot/hivekit/go/modules/authn/internal/store"
)

// AuthnService is a module that manages clients and issues authentication tokens.
//
// This implements IHiveModule and IAuthnModule interfaces and is facade for the
// account store and authenticator.
type AuthnService struct {
	*modules.HiveModuleBase

	config authn.AuthnConfig

	authnStore authnstore.IAuthnStore

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

// Change the password of a client
func (m *AuthnService) SetPassword(clientID string, password string) error {
	return m.authnStore.SetPassword(clientID, password)
}

// Change the role of a client
func (m *AuthnService) SetRole(clientID string, role string) error {
	return m.authnStore.SetRole(clientID, role)
}

// Start the authentication module and handle for login and token refresh requests.
//
// Opens the password store and starts the session manager instance.
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

	return err
}

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

// Create a new authentication service.
//
// authnConfig contains the password storage and token management configuration
func NewAuthnService(authnConfig authn.AuthnConfig) *AuthnService {

	passwordFile := authnConfig.PasswordFile
	encryption := authnConfig.Encryption
	authnStore := authnstore.NewAuthnFileStore(passwordFile, encryption)
	sessionManager := NewSessionManager(authnStore, authnConfig.KeysDir)

	// this module is a singleton that exposes multiple service things
	thingID := authn.AuthnServiceModuleType
	m := &AuthnService{
		HiveModuleBase: modules.NewHiveModuleBase(thingID, 0),
		config:         authnConfig,
		authnStore:     authnStore,
		sessionManager: sessionManager,
		// sessionStart: make(map[string]time.Time),
	}
	var _ modules.IHiveModule = m // interface check
	var _ authn.IAuthnService = m // interface check
	return m
}

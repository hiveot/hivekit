package serviceimpl

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/cells"
	"github.com/hiveot/hivekit/go/cells/authn"
	authnstore "github.com/hiveot/hivekit/go/cells/authn/internal/store"
)

// AuthnServiceImpl manages client accounts and issues authentication tokens.
//
// This implements IHiveCell and IAuthn interfaces and is facade for the account store and authenticator.
type AuthnServiceImpl struct {
	*cells.HiveCellBase

	config authn.AuthnConfig

	authnStore authnstore.IAuthnStore

	// Creation and validation of session tokens
	sessionManager *SessionManager
}

// AddClient adds a client. This fails if the client already exists
// This should only be usable by administrators.
func (svc *AuthnServiceImpl) AddClient(clientID string, displayName string, role string) error {

	_, err := svc.authnStore.GetProfile(clientID)
	if err == nil {
		return fmt.Errorf("Account for client '%s' already exists", clientID)
	}

	newProfile := authn.ClientProfile{
		ClientID:    clientID,
		DisplayName: displayName,
		Role:        role,
	}
	return svc.authnStore.Add(newProfile)
}

// GetProfile return the client's profile
func (svc *AuthnServiceImpl) GetProfile(clientID string) (profile authn.ClientProfile, err error) {
	return svc.authnStore.GetProfile(clientID)
}

// GetProfile return a list of client profiles
func (svc *AuthnServiceImpl) GetProfiles() (profiles []authn.ClientProfile, err error) {
	return svc.authnStore.GetProfiles()
}

func (svc *AuthnServiceImpl) GetSessionManager() authn.ISessionManager {
	return svc.sessionManager
}

// Handle requests to cells of this service
func (svc *AuthnServiceImpl) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) error {

	switch req.ThingID {
	case authn.AuthnAdminServiceID:
		return HandleAuthnAdminRequest(svc, req, replyTo)
	case authn.AuthnUserServiceID:
		return HandleAuthnUserRequest(svc, req, replyTo)
	default:
		// forward
		return svc.HiveCellBase.HandleRequest(req, replyTo)
	}
}

// Remove a client
func (svc *AuthnServiceImpl) RemoveClient(clientID string) error {
	return svc.authnStore.Remove(clientID)
}

// Save the token to the keys directory under the name {clientID}.token
//
// Intended for storing tokens for core services and admin user.
// For internal use only. This might change in the future
func (svc *AuthnServiceImpl) SaveToken(clientID string, token string) error {
	tokenFile := filepath.Join(svc.config.KeysDir, clientID+api.DefaultTokenFileSuffix)

	err := os.MkdirAll(svc.config.KeysDir, 0700)
	if err != nil {
		slog.Error("SaveToken can't ensure directory exist.",
			"keysdir", svc.config.KeysDir, "err", err.Error())
	}
	// the old token can't be overwritten
	_ = os.Remove(tokenFile)
	err = os.WriteFile(tokenFile, []byte(token), 0400)
	if err != nil {
		slog.Error("SaveToken failed", "err", err.Error())
	}

	return err
}

// Change the password of a client
func (svc *AuthnServiceImpl) SetPassword(clientID string, password string) error {
	return svc.authnStore.SetPassword(clientID, password)
}

// Change the role of a client
func (svc *AuthnServiceImpl) SetRole(clientID string, role string) error {
	return svc.authnStore.SetRole(clientID, role)
}

// Start the authentication service and handle for login and token refresh requests.
//
// This opens the password store and starts the session manager instance.
//
// If a validity period is set for an administrator then create a new token file
// for this administrator. {adminID}.token
func (svc *AuthnServiceImpl) Start() (err error) {

	slog.Info("Start: Starting authn")
	err = svc.authnStore.Open()
	if err != nil {
		return err
	}

	err = svc.sessionManager.Start()
	if err != nil {
		return err
	}
	// ensure the administrator account exists
	if svc.config.AdminUserID != "" {
		_, err = svc.GetProfile(svc.config.AdminUserID)
		if err != nil {
			err = svc.AddClient(svc.config.AdminUserID, "Administrator", authn.ClientRoleAdmin)

			if err == nil && svc.config.AdminTokenValidityDays > 0 {
				validity := time.Duration(svc.config.AdminTokenValidityDays) * 24 * time.Hour
				// create a new token for this session
				adminToken, _, _ := svc.sessionManager.CreateToken(svc.config.AdminUserID, validity)
				err = svc.SaveToken(svc.config.AdminUserID, adminToken)
			}
		}
	}
	return err
}

// Stop closes the client store and releases resources
func (svc *AuthnServiceImpl) Stop() {
	slog.Info("Stop: Stopping authn")
	svc.authnStore.Close()
}

// UpdateProfile update the client profile
// only administrators are allowed to update the role
func (svc *AuthnServiceImpl) UpdateProfile(senderID string, newProfile authn.ClientProfile) error {
	senderProf, err := svc.authnStore.GetProfile(senderID)
	if err != nil {
		return fmt.Errorf("Unknown sender '%s'", senderID)
	}
	clientProf, err := svc.authnStore.GetProfile(newProfile.ClientID)
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
	return svc.authnStore.UpdateProfile(newProfile)
}

// Create a new authentication service.
//
// authnConfig contains the password storage and token management configuration
func NewAuthnServiceImpl(authnConfig authn.AuthnConfig) *AuthnServiceImpl {

	passwordFile := authnConfig.PasswordFile
	encryption := authnConfig.Encryption
	authnStore := authnstore.NewAuthnFileStore(passwordFile, encryption)
	sessionManager := NewSessionManager(authnStore, authnConfig.KeysDir)

	// this service is a singleton that exposes multiple service things
	thingID := authn.AuthnServiceCellType
	svc := &AuthnServiceImpl{
		HiveCellBase:   cells.NewHiveCellBase(thingID, 0),
		config:         authnConfig,
		authnStore:     authnStore,
		sessionManager: sessionManager,
		// sessionStart: make(map[string]time.Time),
	}
	var _ api.IHiveCell = svc       // interface check
	var _ authn.IAuthnService = svc // interface check
	return svc
}

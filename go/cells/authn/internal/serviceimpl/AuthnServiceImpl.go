package serviceimpl

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/cells/authn"
	authnstore "github.com/hiveot/hivekit/go/cells/authn/internal/store"
	directory_service "github.com/hiveot/hivekit/go/cells/directory/service"
	"github.com/hiveot/hivekit/go/cells/thing"
	"github.com/hiveot/hivekit/go/utils"
)

// AuthnServiceImpl manages client accounts and issues authentication tokens.
//
// This implements IHiveCell and IAuthn interfaces and is facade for the account store and authenticator.
type AuthnServiceImpl struct {
	*thing.ExposedThing

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
	err = svc.authnStore.Add(newProfile)
	// last track props
	svc.PubProperty(svc.GetThingID(), authn.AdminPropNrClients, svc.authnStore.Count(), false)

	return err
}

// Create the admin account if it doesn't exist and create a new auth token
func (svc *AuthnServiceImpl) CreateAdminAccount() error {
	_, err := svc.GetProfile(svc.config.AdminUserID)
	if err != nil {
		err = svc.AddClient(svc.config.AdminUserID, "Administrator", authn.ClientRoleAdmin)

		if err == nil && svc.config.AdminTokenValidityDays > 0 {
			validity := time.Duration(svc.config.AdminTokenValidityDays) * 24 * time.Hour
			// create a new token for this session
			adminToken, _, _ := svc.sessionManager.CreateToken(svc.config.AdminUserID, validity)
			err = svc.SaveToken(svc.config.AdminUserID, adminToken)
		}
	}
	return err
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

// Handle the authn service administration requests
// This should only be authorized for administrators.
func (svc *AuthnServiceImpl) HandleServiceRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) error {
	var output any
	var err error

	if req.Operation == td.OpInvokeAction {
		switch req.Name {

		case authn.AdminActionAddClient:
			args := authn.AdminAddClientArgs{}
			err = utils.DecodeAsObject(req.Input, &args)
			if err == nil {
				err = svc.AddClient(args.ClientID, args.DisplayName, args.Role)
			}
		case authn.AdminActionGetProfile:
			var clientID string
			err = utils.DecodeAsObject(req.Input, &clientID)
			if err == nil {
				output, err = svc.GetProfile(clientID)
			}
		case authn.AdminActionGetProfiles:
			output, err = svc.GetProfiles()
		case authn.AdminActionRemoveClient:
			var clientID string
			err = utils.DecodeAsObject(req.Input, &clientID)
			if err == nil {
				err = svc.RemoveClient(clientID)
			}
		case authn.AdminActionSetPassword:
			var args authn.AdminSetPasswordArgs // same as user
			err = utils.DecodeAsObject(req.Input, &args)
			if err == nil {
				err = svc.SetPassword(args.UserName, args.Password)
			}
		case authn.AdminActionUpdateProfile:
			var profile authn.ClientProfile
			err = utils.DecodeAsObject(req.Input, &profile)
			if err == nil {
				err = svc.UpdateProfile(req.SenderID, profile)
			}
		default:
			err = errors.New("Unknown action '" + req.Name + "' for service '" + req.ThingID + "'")
		}
		resp := req.CreateResponse(output, err)
		replyTo(resp)
	} else {
		// not an action. let the exposed thing base handle it
		err = svc.ExposedThing.HandleRequest(req, replyTo)
	}
	return err
}

// Handle requests to cells of this service
func (svc *AuthnServiceImpl) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) error {

	// two TDs == two exposed thing services - one for admin-only
	// option 1: implement as separate cells
	// option 2: implement as a single cell with separate request handlers

	switch req.ThingID {
	case authn.AuthnAdminServiceID:
		return svc.HandleServiceRequest(req, replyTo)
	case authn.AuthnUserServiceID:
		return HandleAuthnUserRequest(svc, req, replyTo)
	default:
		// forward
		return svc.HiveCellBase.HandleRequest(req, replyTo)
	}
}

// Publish the service admin and user service TDs to the directory.
func (svc *AuthnServiceImpl) PublishTD() error {
	adminTM := authn.AuthnServiceTD
	userTM := authn.AuthnUserTD
	reqSink := svc.GetRequestSink()
	if reqSink == nil {
		return fmt.Errorf("PublishTD: No request sink set.")
	}
	err := directory_service.UpdateTD("", string(adminTM), reqSink.HandleRequest)
	if err == nil {
		err = directory_service.UpdateTD("", string(userTM), reqSink.HandleRequest)
	}
	return err
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

// Set the request sink and publish the service TD
func (svc *AuthnServiceImpl) SetRequestSink(reqSink api.IHiveCell) {
	svc.HiveCellBase.SetRequestSink(reqSink)
	svc.PublishTD()
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
// This uses thingID authn.AuthnAdminServiceID
//
// authnConfig contains the password storage and token management configuration
func StartAuthnServiceImpl(authnConfig authn.AuthnConfig) (*AuthnServiceImpl, error) {

	slog.Info("Start: Starting authn")
	passwordFile := authnConfig.PasswordFile
	encryption := authnConfig.Encryption
	authnStore := authnstore.NewAuthnFileStore(passwordFile, encryption)
	err := authnStore.Open()
	if err != nil {
		return nil, err
	}

	sessionManager, err := StartSessionManager(authnStore, authnConfig.KeysDir)
	if err != nil {
		return nil, err
	}

	// this service is the admin service that also exposes the user service service thing
	svc := &AuthnServiceImpl{
		ExposedThing:   thing.StartExposedThing(authn.AuthnAdminServiceID, nil),
		config:         authnConfig,
		authnStore:     authnStore,
		sessionManager: sessionManager,
	}
	// update the readable properties
	svc.SetProperty(svc.GetThingID(), authn.AdminPropNrClients, authnStore.Count())

	// ensure the administrator account exists
	if svc.config.AdminUserID != "" {
		err = svc.CreateAdminAccount()
	}

	var _ api.IHiveCell = svc       // interface check
	var _ authn.IAuthnService = svc // interface check
	return svc, err
}

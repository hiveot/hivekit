package internal

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/cells/authn"
	"github.com/hiveot/hivekit/go/cells/thing"
	"github.com/hiveot/hivekit/go/utils"
)

// AuthnAdminServiceImpl manages client accounts and issues authentication tokens.
//
// This implements IHiveCell and IAuthn interfaces and is facade for the account store and authenticator.
type AuthnUserServiceImpl struct {
	*thing.ExposedThing

	// passwdDir string

	authnStore authn.IAuthnStore

	// Creation and validation of session tokens
	sessionManager *SessionManager
}

// GetProfile return the client's profile
func (svc *AuthnUserServiceImpl) GetProfile(clientID string) (profile authn.ClientProfile, err error) {
	return svc.authnStore.GetProfile(clientID)
}

func (svc *AuthnUserServiceImpl) GetSessionManager() authn.ISessionManager {
	return svc.sessionManager
}

// Handle requests to cells of this service
func (svc *AuthnUserServiceImpl) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) error {
	var output any
	var err error

	if req.ThingID != svc.GetID() {
		return svc.ForwardRequest(req, replyTo)
	}

	if req.Operation == td.OpInvokeAction {
		switch req.Name {

		case authn.UserActionGetProfile:
			if err == nil {
				output, err = svc.GetProfile(req.SenderID)
			} else {
				err = errors.New("bad function argument: " + err.Error())
			}
		case authn.UserActionLogout:
			sessMgr := svc.GetSessionManager()
			sessMgr.Logout(req.SenderID)

		case authn.UserActionRefreshToken:
			var oldToken string
			err = utils.DecodeAsObject(req.Input, &oldToken)
			if err == nil {
				sessMgr := svc.GetSessionManager()
				output, _, err = sessMgr.RefreshToken(req.SenderID, oldToken)
			} else {
				err = errors.New("bad function argument: " + err.Error())
			}

		case authn.UserActionSetPassword:
			var newPassword string
			err = utils.DecodeAsObject(req.Input, &newPassword)
			if err == nil {
				err = svc.SetPassword(req.SenderID, newPassword)
			} else {
				err = errors.New("bad function argument: " + err.Error())
			}

		case authn.UserActionUpdateProfile:
			var profile authn.ClientProfile

			err = utils.DecodeAsObject(req.Input, &profile)
			if err == nil {
				err = svc.UpdateProfile(req.SenderID, profile)
			} else {
				err = errors.New("bad function argument: " + err.Error())
			}
		default:
			err = errors.New("Unknown action '" + req.Name + "' for service '" + req.ThingID + "'")
		}
		resp := req.CreateResponse(output, err)
		err = replyTo(resp)
	} else {
		// not an action. let the exposed thing base handle it
		err = svc.ExposedThing.HandleRequest(req, replyTo)
	}
	return err
}

// Change the password of a user
func (svc *AuthnUserServiceImpl) SetPassword(clientID string, password string) error {
	return svc.authnStore.SetPassword(clientID, password)
}

// Start publishes the service TD to the directory or discovery.
func (svc *AuthnUserServiceImpl) Start() {
	tdoc, _ := td.UnmarshalTD(string(authn.AuthnUserTD))
	tdoc.ID = authn.AuthnUserServiceThingID
	tdoc.SetType(authn.AuthnUserServiceThingID)
	userTD := tdoc.ToString()
	_ = svc.PublishTD(userTD)
}

// Stop closes the client store and releases resources
func (svc *AuthnUserServiceImpl) Stop() {
	slog.Info("Stop: Stopping authn")
	svc.authnStore.Close()
}

// UpdateProfile updates the client profile.
// The role will remain unchanged unless the user is an admin.
func (svc *AuthnUserServiceImpl) UpdateProfile(senderID string, newProfile authn.ClientProfile) error {
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

// Create a ready-to-use authentication service.
//
// This uses cellID authn.AuthnUserServiceID
// Call Start() to publish its TD.
//
//	tokensDir is location to store auth tokens
//	authnStore account store
//	createAdminToken flag, create a new token for the default admin account
func NewAuthnUserServiceImpl(
	tokensDir string, authnStore authn.IAuthnStore, createAdminToken bool) (
	*AuthnUserServiceImpl, error) {

	slog.Info("NewAuthnUserServiceImpl: creating authn user service")

	sessionManager, err := StartSessionManager(authnStore, tokensDir)
	if err != nil {
		return nil, err
	}

	// this service is the admin service that also exposes the user service service thing
	svc := &AuthnUserServiceImpl{
		ExposedThing:   thing.NewExposedThing(authn.AuthnUserServiceThingID, nil),
		authnStore:     authnStore,
		sessionManager: sessionManager,
	}
	// ensure the administrator account has a token
	if createAdminToken {
		adminID := api.DefaultAdminUserID
		validity := authn.AdminTokenValidityDays * 24 * time.Hour
		adminToken, _, _ := svc.sessionManager.CreateToken(adminID, validity)
		err = svc.sessionManager.SaveToken(adminID, adminToken)
	}

	var _ api.IHiveCell = svc           // interface check
	var _ authn.IAuthnUserService = svc // interface check
	return svc, err
}

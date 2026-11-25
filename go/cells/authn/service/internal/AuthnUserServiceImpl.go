package internal

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/wot"
	"github.com/hiveot/hivekit/go/cells/authn"
	"github.com/hiveot/hivekit/go/cells/thing"
	"github.com/hiveot/hivekit/go/utils"
)

// AuthnAdminServiceImpl manages client accounts and issues authentication tokens.
//
// This implements IHiveCell and IAuthn interfaces and is facade for the account store and authenticator.
type AuthnUserServiceImpl struct {
	*thing.ExposedThing

	authnStore authn.IAuthnStore

	// Creation and validation of session tokens
	sessionManager *SessionManager
}

// GetProfile return the client's profile
func (svc *AuthnUserServiceImpl) GetProfile(clientID string) (profile authn.ClientProfile, err error) {
	return svc.authnStore.GetProfile(clientID)
}

// Handle requests to cells of this service
func (svc *AuthnUserServiceImpl) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) error {
	var output any
	var err error

	if req.ThingID != svc.GetID() && req.ThingID != authn.AuthnUserServiceCellType {
		return svc.ForwardRequest(req, replyTo)
	}
	slog.Info("HandleRequest", slog.String("name", req.Name), slog.String("sender", req.SenderID))

	if req.Operation == wot.OpInvokeAction {
		switch req.Name {

		case authn.GetProfileUserAction:
			if err == nil {
				output, err = svc.GetProfile(req.SenderID)
			} else {
				err = errors.New("bad function argument: " + err.Error())
			}
		case authn.LoginUserAction:
			var args authn.UserLoginArgs
			err = req.DecodeInput(&args)
			if err == nil {
				output, err = svc.Login(args.UserName, args.Password)
			}

		case authn.LogoutUserAction:
			svc.sessionManager.Logout(req.SenderID)

		case authn.RefreshTokenUserAction:
			var oldToken string
			err = utils.DecodeAsObject(req.Input, &oldToken)
			if err == nil {
				output, _, err = svc.sessionManager.RefreshToken(req.SenderID, oldToken)
			} else {
				err = errors.New("bad function argument: " + err.Error())
			}

		case authn.SetPasswordUserAction:
			var newPassword string
			err = utils.DecodeAsObject(req.Input, &newPassword)
			if err == nil {
				err = svc.SetPassword(req.SenderID, newPassword)
			} else {
				err = errors.New("bad function argument: " + err.Error())
			}

		case authn.UpdateProfileUserAction:
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

// FIXME: change to use passkey, eg LoginWithPasskey(signedChallenge)
// eg, server sends a challenge; client signs it; server verifies using pubkey
func (svc *AuthnUserServiceImpl) Login(clientID string, password string) (token string, err error) {
	token, _, err = svc.sessionManager.Login(clientID, password)
	if err == nil {
		svc.authnStore.SetLogin(clientID, utils.FormatNowUTCMilli())
	}
	return token, err
}

// Change the password of a user
func (svc *AuthnUserServiceImpl) SetPassword(clientID string, password string) error {
	return svc.authnStore.SetPassword(clientID, password)
}

// Start publishes the service TD to the directory or discovery.
func (svc *AuthnUserServiceImpl) Start() {

	tdoc, _ := wot.UnmarshalTD(string(authn.AuthnUserTD))
	tdoc.ID = svc.GetID()
	tdoc.SetType(authn.AuthnUserServiceCellType)
	_ = svc.WriteTD(tdoc)
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
		if senderProf.Role != api.ClientRoleAdmin && senderProf.Role != api.ClientRoleService {
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
//	sessionManager that manages auth tokens
//	authnStore account store
func NewAuthnUserServiceImpl(
	authnStore authn.IAuthnStore, sessionManager *SessionManager) (
	*AuthnUserServiceImpl, error) {

	var err error
	slog.Info("NewAuthnUserServiceImpl: creating authn user service")

	// this service is the admin service that also exposes the user service service thing
	svc := &AuthnUserServiceImpl{
		ExposedThing:   thing.NewExposedThing(authn.AuthnUserServiceCellType, nil),
		authnStore:     authnStore,
		sessionManager: sessionManager,
	}

	var _ api.IHiveCell = svc           // interface check
	var _ authn.IAuthnUserService = svc // interface check
	return svc, err
}

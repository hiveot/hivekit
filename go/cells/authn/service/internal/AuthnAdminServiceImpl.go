package internal

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

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
type AuthnAdminServiceImpl struct {
	*thing.ExposedThing

	authnStore authn.IAuthnStore

	sessionManager authn.ISessionManager
}

// AddClient adds a client. This fails if the client already exists
// This should only be usable by administrators.
func (svc *AuthnAdminServiceImpl) AddClient(clientID string, displayName string, role string) error {

	_, err := svc.authnStore.GetProfile(clientID)
	if err == nil {
		return fmt.Errorf("Account for client '%s' already exists", clientID)
	}

	newProfile := authn.ClientProfile{
		ClientID:    clientID,
		DisplayName: displayName,
		Role:        role,
		TimeCreated: time.Now().Format(time.RFC3339),
	}
	err = svc.authnStore.Add(newProfile)
	// last track props
	svc.PubProperty(svc.GetID(), authn.AdminPropNrClients, svc.authnStore.Count(), false)

	return err
}

// GetProfile return the client's profile
func (svc *AuthnAdminServiceImpl) GetProfile(clientID string) (profile authn.ClientProfile, err error) {
	return svc.authnStore.GetProfile(clientID)
}

// GetProfile return a list of client profiles
func (svc *AuthnAdminServiceImpl) GetProfiles() (profiles []authn.ClientProfile, err error) {
	return svc.authnStore.GetProfiles()
}

// Handle requests to cells of this service
func (svc *AuthnAdminServiceImpl) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) error {
	var output any
	var err error

	if req.ThingID != svc.GetID() {
		return svc.ForwardRequest(req, replyTo)
	}

	if req.Operation == wot.OpInvokeAction {
		switch req.Name {

		case authn.AdminAddClientAction:
			args := authn.AdminAddClientInput{}
			err = utils.DecodeAsObject(req.Input, &args)
			if err == nil {
				err = svc.AddClient(args.ClientID, args.DisplayName, args.Role)
			}
		case authn.AdminCreateTokenAction:
			var clientID string
			err = req.DecodeInput(&clientID)
			if err == nil {
				output, _, err = svc.sessionManager.CreateToken(clientID, 0)
			}
		case authn.AdminGetProfileAction:
			var clientID string
			err = req.DecodeInput(&clientID)
			if err == nil {
				output, err = svc.GetProfile(clientID)
			}
		case authn.AdminGetProfilesAction:
			output, err = svc.GetProfiles()

		case authn.AdminRemoveClientAction:
			var clientID string
			err = utils.DecodeAsObject(req.Input, &clientID)
			if err == nil {
				err = svc.RemoveClient(clientID)
			}
		case authn.AdminSetPasswordAction:
			var args authn.AdminSetPasswordInput // same as user
			err = utils.DecodeAsObject(req.Input, &args)
			if err == nil {
				err = svc.SetPassword(args.UserName, args.Password)
			}
		case authn.AdminSetRoleAction:
			var args authn.AdminSetRoleInput
			err = utils.DecodeAsObject(req.Input, &args)
			if err == nil {
				err = svc.SetRole(args.ClientID, args.Role)
			}
		case authn.AdminUpdateProfileAction:
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

// Remove a client
func (svc *AuthnAdminServiceImpl) RemoveClient(clientID string) error {
	return svc.authnStore.Remove(clientID)
}

// Change the password of a client
func (svc *AuthnAdminServiceImpl) SetPassword(clientID string, password string) error {
	return svc.authnStore.SetPassword(clientID, password)
}

// Change the role of a client
func (svc *AuthnAdminServiceImpl) SetRole(clientID string, role string) error {
	return svc.authnStore.SetRole(clientID, role)
}

// Start publishes the service admin service TDs to the directory or discovery.
func (svc *AuthnAdminServiceImpl) Start() {
	tdoc, _ := wot.UnmarshalTD(string(authn.AuthnAdminTD))
	tdoc.ID = authn.DefaultAuthnAdminThingID
	tdoc.SetType(authn.AuthnAdminServiceType)
	_ = svc.WriteTD(tdoc)
}

// Stop closes the client store and releases resources
func (svc *AuthnAdminServiceImpl) Stop() {
	slog.Info("Stop: Stopping authn")
	svc.authnStore.Close()
}

// UpdateProfile update the client profile
// only administrators are allowed to update the role
func (svc *AuthnAdminServiceImpl) UpdateProfile(senderID string, newProfile authn.ClientProfile) error {
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
// This uses thingID authn.AuthnAdminServiceID
// Call Start() to publish its TD.
//
//	authUserSvc user services including storage and session management
//	createAdminAcct set to create a default admin user and token
func NewAuthnAdminServiceImpl(
	authnStore authn.IAuthnStore,
	sessionManager authn.ISessionManager,
	createAdminAcct bool) (*AuthnAdminServiceImpl, error) {
	var err error

	slog.Info("NewAuthnServiceImpl: creating authn service")

	// this service is the admin service that also exposes the user service service thing
	svc := &AuthnAdminServiceImpl{
		ExposedThing:   thing.NewExposedThing(authn.DefaultAuthnAdminThingID, nil),
		authnStore:     authnStore,
		sessionManager: sessionManager,
	}
	// update the readable properties
	svc.SetProperty(svc.GetID(), authn.AdminPropNrClients, authnStore.Count())

	// ensure the administrator account exists
	if createAdminAcct {
		adminID := api.DefaultAdminUserID
		_, err = svc.GetProfile(adminID)
		if err != nil {
			err = svc.AddClient(adminID, "Administrator", api.ClientRoleAdmin)
		}
	}

	var _ api.IHiveCell = svc            // interface check
	var _ authn.IAuthnAdminService = svc // interface check
	return svc, err
}

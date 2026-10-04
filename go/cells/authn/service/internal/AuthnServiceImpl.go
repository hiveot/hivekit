package internal

import (
	"path/filepath"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/cells/authn"
	authn_filestore "github.com/hiveot/hivekit/go/cells/authn/filestore"
	"github.com/hiveot/hivekit/go/cells/thing"
)

// AuthnAdminServiceImpl manages client accounts and issues authentication tokens.
//
// This implements IHiveCell and IAuthn interfaces and is facade for the account store and authenticator.
type AuthnServiceImpl struct {
	*thing.ExposedThing

	authnStore authn.IAuthnStore
	adminSvc   *AuthnAdminServiceImpl
	userSvc    *AuthnUserServiceImpl
}

// Return the admin management service
func (svc *AuthnServiceImpl) GetAdminService() authn.IAuthnAdminService {
	return svc.adminSvc
}

// Return the user self management service
func (svc *AuthnServiceImpl) GetUserService() authn.IAuthnUserService {
	return svc.userSvc
}

func (svc *AuthnServiceImpl) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) error {
	if req.ThingID == svc.userSvc.GetID() {
		return svc.userSvc.HandleRequest(req, replyTo)
	} else if req.ThingID == svc.adminSvc.GetID() {
		return svc.adminSvc.HandleRequest(req, replyTo)
	} else {
		return svc.ForwardRequest(req, replyTo)
	}
}

// Set the handler of requests for admin and user services
// Intended for publishing TD's on start.
func (svc *AuthnServiceImpl) SetRequestSink(sink api.IHiveCell) {
	// this is needed to forward requests not intended for authn
	svc.ExposedThing.SetRequestSink(sink)

	// pass requests from the nested services to this sink, eg publish TD
	svc.adminSvc.SetRequestSink(sink)
	svc.userSvc.SetRequestSink(sink)
}

// Start publishes the service TDs to the directory or discovery.
func (svc *AuthnServiceImpl) Start() {
	// svc.authnStore.Start()
	svc.adminSvc.Start()
	svc.userSvc.Start()
}

// Stop the admin and storage services
func (svc *AuthnServiceImpl) Stop() {
	svc.authnStore.Close()
	svc.adminSvc.Stop()
	svc.userSvc.Stop()
}

// Create a ready-to-use authentication service.
//
// This service provides both admin and user services.
// Call Start() to publish ther TDs.
//
//	tokensDir with the location to store auth tokens
//	storageDir with the location to store the and accounts file
//	createAdmin flag, create a default admin account and new token
func NewAuthnServiceImpl(tokensDir string, storageDir string, createAdmin bool) (
	*AuthnServiceImpl, error) {

	passwdFile := filepath.Join(storageDir, authn.DefaultPasswordFile)
	hashAlgo := authn.PWHASH_ARGON2id

	authnStore, err := authn_filestore.OpenAuthnFileStore(passwdFile, hashAlgo)
	if err != nil {
		return nil, err
	}

	adminSvc, err := NewAuthnAdminServiceImpl(authnStore, createAdmin)
	if err != nil {
		return nil, err
	}
	userSvc, err := NewAuthnUserServiceImpl(tokensDir, authnStore, createAdmin)
	if err != nil {
		return nil, err
	}
	// the authn service wraps the nested services and acts as their proxy
	// FIXME: The ThingID of this cell is dynamic. Can't use this in a star formation...
	// option 1: use admin svc or user svc as ID
	authnSvc := &AuthnServiceImpl{
		ExposedThing: thing.NewExposedThing("", nil),
		authnStore:   authnStore,
		adminSvc:     adminSvc,
		userSvc:      userSvc,
	}
	adminSvc.SetNotificationSink(authnSvc)
	userSvc.SetNotificationSink(authnSvc)
	return authnSvc, nil
}

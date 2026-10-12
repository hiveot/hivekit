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

	authnStore     authn.IAuthnStore
	adminSvc       *AuthnAdminServiceImpl
	sessionManager *SessionManager
	userSvc        *AuthnUserServiceImpl
}

// Return the admin management service
func (svc *AuthnServiceImpl) GetAdminService() authn.IAuthnAdminService {
	return svc.adminSvc
}

func (svc *AuthnServiceImpl) GetSessionManager() authn.ISessionManager {
	return svc.sessionManager
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
	svc.sessionManager.Stop()
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

	sessionManager, err := StartSessionManager(authnStore, tokensDir)
	if err != nil {
		return nil, err
	}

	adminSvc, err := NewAuthnAdminServiceImpl(authnStore, sessionManager, createAdmin)
	if err != nil {
		return nil, err
	}
	userSvc, err := NewAuthnUserServiceImpl(authnStore, sessionManager, createAdmin)
	if err != nil {
		return nil, err
	}
	// the authn service wraps the nested services and acts as their proxy
	// Note: This cell doesnt publish a TD (its admin/user embedded cells do), it cant be
	// used in a star formation. Star formation only supports addressing members directly,
	// but not their embedded things.
	// If this becomes an issue then maybe add 'hasThing(id)' to the cell base. Trying to
	// keep the base small though.
	authnSvc := &AuthnServiceImpl{
		ExposedThing:   thing.NewExposedThing("authnService", nil),
		authnStore:     authnStore,
		adminSvc:       adminSvc,
		sessionManager: sessionManager,
		userSvc:        userSvc,
	}
	adminSvc.SetNotificationSink(authnSvc)
	userSvc.SetNotificationSink(authnSvc)
	return authnSvc, nil
}

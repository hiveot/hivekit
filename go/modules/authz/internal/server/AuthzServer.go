package authzserver

import (
	"fmt"
	"log/slog"

	"github.com/hiveot/hivekit/go/modules"
	authzapi "github.com/hiveot/hivekit/go/modules/authz/api"
	"github.com/hiveot/hivekit/go/msg"
)

// AuthzServer is a module for role based authorization of requests.
//
// This implements IHiveModule and IAuthzModule interfaces and is facade for the
// authorization store. This uses the authenticator provided client role as the role
// for RBAC.
type AuthzServer struct {
	modules.HiveModuleBase

	// config authz.AuthzConfig

	// the handler that provides the client's role
	getRoleHandler func(clientID string) (role string, err error)
}

// Handle requests to be served by this module and filter unauthorized requests.
// This depends on a validated SenderID in the request message.
func (m *AuthzServer) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) error {

	hasPermission := m.HasPermission(req)
	if !hasPermission {
		return fmt.Errorf("Insufficient permissions for request '%s' by client '%s'", req.Operation, req.SenderID)
	}

	if req.ThingID == AuthzAdminServiceID {
		return HandleAuthzAdminRequest(m, req, replyTo)
	}
	// forward the request to the chain
	return m.HiveModuleBase.HandleRequest(req, replyTo)
}

// start opens the store with authorization rules
// currently the RBAC is hard coded so nothing to configure
func (m *AuthzServer) Start(yamlConfig string) (err error) {
	if m.getRoleHandler == nil {
		slog.Warn("AuthzModule: no getRoleHandler provided, only read requests will be accepted")
	}
	m.SetModuleID(authzapi.DefaultAuthzModuleID)
	return nil
}

// Stop closes the rules store and releases resources
func (m *AuthzServer) Stop() {
}

// Create a new instance of the authorization server module.
// The getRole handler is used to determine a client's role for RBAC
func NewAuthzServer(getRoleHandler func(clientID string) (role string, err error)) *AuthzServer {
	m := &AuthzServer{
		getRoleHandler: getRoleHandler,
	}
	var _ modules.IHiveModule = m // check interface
	return m
}

package internal

import (
	"fmt"
	"log/slog"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/cells"
	"github.com/hiveot/hivekit/go/cells/authz"
)

// AuthzServiceImpl is a service for role based authorization of requests.
//
// This implements IHiveCell and IAuthzService interfaces and is facade for the
// authorization store. This uses the authenticator provided client role as the role
// for RBAC.
type AuthzServiceImpl struct {
	*cells.HiveCellBase

	// config authz.AuthzConfig

	// the handler that provides the client's role
	getRoleHandler func(clientID string) (role string, err error)
}

// Handle requests to be served by this service and filter unauthorized requests.
// This depends on a validated SenderID in the request message.
func (svc *AuthzServiceImpl) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) error {

	hasPermission := svc.HasPermission(req)
	if !hasPermission {
		return fmt.Errorf("HandleRequest: Insufficient permissions for request '%s' by client '%s'", req.Operation, req.SenderID)
	}

	if req.ThingID == AuthzAdminServiceID {
		return HandleAuthzAdminRequest(svc, req, replyTo)
	}
	// forward the request to the chain
	return svc.HiveCellBase.HandleRequest(req, replyTo)
}

// Stop closes the rules store and releases resources
func (svc *AuthzServiceImpl) Stop() {
	slog.Info("Stop: Stopping authz")
}

// Start a new instance of the authorization service.
// The getRole handler is used to determine a client's role for RBAC
func StartAuthzServiceImpl(getRoleHandler func(clientID string) (role string, err error)) *AuthzServiceImpl {
	slog.Info("Starting authz")
	// this service is a singleton that exposes multiple service things
	thingID := authz.AuthzServiceCellType
	svc := &AuthzServiceImpl{
		HiveCellBase:   cells.NewHiveCellBase(thingID, 0),
		getRoleHandler: getRoleHandler,
	}
	if getRoleHandler == nil {
		slog.Warn("NewAuthzServiceImpl: no getRoleHandler provided, only read requests will be accepted")
	}

	// currently the RBAC is hard coded so nothing to configure

	var _ api.IHiveCell = svc // check interface
	return svc
}

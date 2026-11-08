package internal

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/wot"
)

// ValidateAuthorization verifies that the sender is authorized for the request.
// Currently this is a simple hard coded RBAC based on the client role.
// Services must handle exceptions to permissions for their own devices/services
// if needed.
//
// This currently hard-codes a basic set of rules:
// 1. viewers can read properties and subscribe to events
// 2. operators can read properties, query and invoke actions
// 3. managers can read properties, write configuration, query and invoke actions
// 4. administrators can do everything
// 5. devices can publish events (notifications) for their own and nested devices
// 6. services can publish events (notifications) for their own service(s) and subscribe to any events
func (svc *AuthzServiceImpl) HasPermission(req *msg.RequestMessage) (hasPermission bool) {
	if svc.getRoleHandler == nil {
		return false
	}
	role, err := svc.getRoleHandler(req.SenderID)
	if err != nil {
		return false // unknown sender
	}

	switch req.Operation {

	// 1. everyone can read properties and subscribe to events
	case wot.OpReadProperty, wot.OpReadMultipleProperties, wot.OpReadAllProperties,
		wot.OpSubscribeEvent, wot.OpSubscribeAllEvents:
		return true
	// 2. operators, managers, administrators and services can also query and invoke actions
	case wot.OpInvokeAction, wot.OpQueryAction, wot.OpQueryAllActions:
		if role == api.ClientRoleOperator ||
			role == api.ClientRoleManager ||
			role == api.ClientRoleAdmin ||
			role == api.ClientRoleService {
			return true
		}
		// devices can update their TD;
		if role == api.ClientRoleDevice {
			return true
		}
	// 3. managers, administrators and services can also write configuration
	case wot.OpWriteProperty, wot.OpWriteMultipleProperties:
		if role == api.ClientRoleManager ||
			role == api.ClientRoleAdmin ||
			role == api.ClientRoleService {
			return true
		}
	// 4. administrators and services can do everything else
	default:
		if role == api.ClientRoleAdmin || role == api.ClientRoleService {
			return true
		}
	}
	return false
}

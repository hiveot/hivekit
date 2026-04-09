package internal

import (
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	authnapi "github.com/hiveot/hivekit/go/modules/authn/api"
)

// ValidateAuthorization verifies that the sender is authorized for the request.
// Currently this is a hard coded RBAC based on the client role. Services must
// handle exceptions to permissions for their own devices/services if needed.
//
// This currently hard-codes a basic set of rules:
// 1. viewers can read properties and subscribe to events
// 2. operators can read properties, query and invoke actions
// 3. managers can read properties, write configuration, query and invoke actions
// 4. administrators can do everything
// 5. agents can publish events (notifications) for their own devices and services
// 6. services can publish events (notifications) for their own devices and services and subscribe to any events
func (m *AuthzService) HasPermission(req *msg.RequestMessage) (hasPermission bool) {
	if m.getRoleHandler == nil {
		return false
	}
	role, err := m.getRoleHandler(req.SenderID)
	if err != nil {
		return false // unknown sender
	}

	// TODO: can the messagefilter be used for configurable rules?
	switch req.Operation {

	// 1. everyone can read properties and subscribe to events
	case td.OpReadProperty, td.OpReadMultipleProperties, td.OpReadAllProperties,
		td.OpSubscribeEvent, td.OpSubscribeAllEvents:
		return true
	// 2. operators, managers, administrators and services can also query and invoke actions
	case td.OpInvokeAction, td.OpQueryAction, td.OpQueryAllActions:
		if role == authnapi.ClientRoleOperator ||
			role == authnapi.ClientRoleManager ||
			role == authnapi.ClientRoleAdmin ||
			role == authnapi.ClientRoleService {
			return true
		}
	// 3. managers, administrators and services can also write configuration
	case td.OpWriteProperty, td.OpWriteMultipleProperties:
		if role == authnapi.ClientRoleManager ||
			role == authnapi.ClientRoleAdmin ||
			role == authnapi.ClientRoleService {
			return true
		}
		// 4. administrators and services can do everything else
	default:
		if role == authnapi.ClientRoleAdmin || role == authnapi.ClientRoleService {
			return true
		}
	}
	return false
}

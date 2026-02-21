package module

import (
	"fmt"

	"github.com/hiveot/hivekit/go/modules/authn"
	"github.com/hiveot/hivekit/go/msg"
	"github.com/hiveot/hivekit/go/wot"
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
func (m *AuthzModule) ValidateAuthorization(req *msg.RequestMessage) (err error) {
	prof, err := m.authn.GetProfile(req.SenderID)
	if err != nil {
		return err
	}
	role := prof.Role

	// TODO: can the messagefilter be used for configurable rules?
	switch req.Operation {

	// 1. everyone can read properties and subscribe to events
	case wot.OpReadProperty, wot.OpReadMultipleProperties, wot.OpReadAllProperties,
		wot.OpSubscribeEvent, wot.OpSubscribeAllEvents:
		return nil
	// 2. operators, managers, administrators and services can also query and invoke actions
	case wot.OpInvokeAction, wot.OpQueryAction, wot.OpQueryAllActions:
		if role == authn.ClientRoleOperator ||
			role == authn.ClientRoleManager ||
			role == authn.ClientRoleAdmin ||
			role == authn.ClientRoleService {
			return nil
		}
	// 3. managers, administrators and services can also write configuration
	case wot.OpWriteProperty, wot.OpWriteMultipleProperties:
		if role == authn.ClientRoleManager ||
			role == authn.ClientRoleAdmin ||
			role == authn.ClientRoleService {
			return nil
		}
		// 4. administrators and services can do everything else
	default:
		if role == authn.ClientRoleAdmin || role == authn.ClientRoleService {
			return nil
		}
	}
	return fmt.Errorf("Client '%s' is unauthorized for operation '%s'", req.SenderID, req.Operation)
}

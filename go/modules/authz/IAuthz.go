package authz

import (
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/modules"
)

// default module type and instance identification
const AuthzModuleType = "authz"

// Authorisation server module for authorizing module requests based on client roles.
type IAuthzService interface {
	modules.IHiveModule

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
	HasPermission(req *msg.RequestMessage) (hasPermission bool)
}

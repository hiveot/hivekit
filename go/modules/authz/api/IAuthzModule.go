package authzapi

import "github.com/hiveot/hivekit/go/modules"

// default module instance identification
const DefaultAuthzModuleID = "authz"

// Authorisation server module for authorizing module requests based on client roles.
type IAuthzServer interface {
	modules.IHiveModule
}

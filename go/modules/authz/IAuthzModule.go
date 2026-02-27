package authz

import "github.com/hiveot/hivekit/go/modules"

// Authorisation module for authorizing module requests based on client roles.
type IAuthzModule interface {
	modules.IHiveModule
}

package authzserver

import (
	"fmt"

	authzapi "github.com/hiveot/hivekit/go/modules/authz/api"
	"github.com/hiveot/hivekit/go/msg"
)

// AuthzAdminServiceID is the thingID of the device/service for administration of the module
const AuthzAdminServiceID = "AuthzAdmin"

// HandleAuthzAdminRequest handles messaging requests to the for administration of the module
func HandleAuthzAdminRequest(m authzapi.IAuthzServer, req *msg.RequestMessage, replyTo msg.ResponseHandler) error {
	return fmt.Errorf("HandleAuthzAdminRequest: nothing to do here")
}

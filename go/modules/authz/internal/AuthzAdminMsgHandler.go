package internal

import (
	"fmt"

	"github.com/hiveot/hivekit/go/api/msg"
	authzapi "github.com/hiveot/hivekit/go/modules/authz/api"
)

// AuthzAdminServiceID is the thingID of the device/service for administration of the module
const AuthzAdminServiceID = "AuthzAdmin"

// HandleAuthzAdminRequest handles messaging requests to the for administration of the module
func HandleAuthzAdminRequest(m authzapi.IAuthzService, req *msg.RequestMessage, replyTo msg.ResponseHandler) error {
	return fmt.Errorf("HandleAuthzAdminRequest: nothing to do here")
}

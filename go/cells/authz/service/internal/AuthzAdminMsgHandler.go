package internal

import (
	"fmt"
	"log/slog"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/cells/authz"
)

// AuthzAdminServiceID is the thingID of the device/service for administration of the service
const AuthzAdminServiceID = "AuthzAdmin"

// HandleAuthzAdminRequest handles messaging requests to the for administration of the service
func HandleAuthzAdminRequest(m authz.IAuthzService, req *msg.RequestMessage, replyTo msg.ResponseHandler) error {
	slog.Info("HandleRequest", slog.String("name", req.Name), slog.String("sender", req.SenderID))
	return fmt.Errorf("HandleAuthzAdminRequest: nothing to do here")
}

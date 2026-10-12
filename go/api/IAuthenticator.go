package api

import (
	"time"

	"github.com/hiveot/hivekit/go/api/td"
)

// Predefined roles of a client
// The roles are hierarchical in permissions:
// Authorization using these roles is applied through an authz service.
// Custom roles can be added if needed but their persmissions need to be
// managed in the authz service.
const (

	// ClientRoleNone means that the client has no permissions.
	// It can not do anything until the role is upgraded to viewer or better
	ClientRoleNone string = "none"

	// ClientRoleViewer for users that can view information for devices/services
	// they have access to.
	// Viewers cannot invoke actions or change configuration.
	ClientRoleViewer string = "viewer"

	// ClientRoleDevice for devices.
	//
	// Devices publish Thing information for itself and possible nested devices,
	// and handle request for those devices.
	ClientRoleDevice string = "device"

	// ClientRoleOperator for users that operate devices and services.
	//
	// Operators can view and control devices/services they have access to but
	// not configure them.
	ClientRoleOperator string = "operator"

	// ClientRoleManager for users that manage devices.
	//
	// Managers can view, control and configure devices/services they have access to.
	ClientRoleManager string = "manager"

	// ClientRoleAdmin for users that administer the system.
	//
	// Administrators can view, control and configure all devices and services.
	ClientRoleAdmin string = "admin"

	// ClientRoleService for Service role
	//
	// Services are equivalent to both an admin user and exposed things.
	ClientRoleService string = "service"
)

// Interface of the authentication capability for setting TD security scheme
// and authenticating incoming connections.

type IAuthenticator interface {

	// AddSecurityScheme adds the wot securityscheme that describes this authenticator to the given TD
	// If the scheme is already included in the TD then it is replaced.
	AddSecurityScheme(tdoc *td.TD)

	// Return the role of the client
	GetRole(clientID string) string

	// ValidateClient verifies the secret is valid for the claimed clientID.
	//
	// This returns the validated clientID and the time the secret was issued and is valid for.
	// This returns an error if the secret is invalid, has expired, or the client is blocked..
	ValidateClient(claimedClientID string, secret string) (clientID string, issuedAt time.Time, validUntil time.Time, err error)
}

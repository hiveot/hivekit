// package authn with admin service messaging definitions
package authn

// property, event and action names
const (
	// Property names
	AdminPropNrClients = "nrClients"

	// Event names
	AdminEventAdded   = "added"
	AdminEventRemoved = "removed"

	// Action names
	AdminAddClientAction     = "addClient"
	AdminCreateTokenAction   = "createToken"
	AdminGetProfileAction    = "getProfile"
	AdminGetProfilesAction   = "getProfiles"
	AdminRemoveClientAction  = "removeClient"
	AdminSetPasswordAction   = "setPassword"
	AdminSetRoleAction       = "setRole"
	AdminUpdateProfileAction = "updateProfile"
)

// args for setting a client's password
// used in http and rrn messaging
type AdminSetPasswordInput struct {
	UserName string `json:"username"`
	Password string `json:"password"`
}

// args for setting a client's role
// used in rrn messaging
type AdminSetRoleInput struct {
	ClientID string `json:"clientID"`
	Role     string `json:"role"`
}

// args for updating a client's profile
// used in http and rrn messaging
type AdminUpdateProfileInput struct {
	ClientID string        `json:"clientID"`
	Profile  ClientProfile `json:"profile"`
}

// AdminAddClientInput defines the arguments of the addClient function
// Add Client - Create an account for IoT devices, services or consumers
type AdminAddClientInput struct {

	// ClientID with Client ID
	ClientID string `json:"clientID,omitempty"`

	// DisplayName with Display Name
	DisplayName string `json:"displayName,omitempty"`

	// Role of the client
	Role string `json:"role,omitempty"`
}

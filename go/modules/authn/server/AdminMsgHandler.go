package authnserver

import (
	"errors"

	"github.com/hiveot/hivekit/go/modules/authn"
	"github.com/hiveot/hivekit/go/msg"
	"github.com/hiveot/hivekit/go/utils"
)

// AdminServiceID is the thingID of the device/service as used by agents.
// Agents use this to publish events and subscribe to actions
const AuthnAdminServiceID = "AuthnAdmin"

// property, event and action names
const (
	// Property names
	AdminPropNrClients = "nrClients"

	// Event names
	AdminEventAdded   = "added"
	AdminEventRemoved = "removed"

	// Action names
	AdminActionAddClient   = "addClient"
	AdminActionGetProfile  = "getProfile"
	AdminActionGetProfiles = "getProfiles"
	// AdminActionGetSessions  = "getSessions"
	AdminActionRemoveClient  = "removeClient"
	AdminActionSetPassword   = "setPassword"
	AdminActionSetRole       = "setRole"
	AdminActionUpdateProfile = "updateProfile"
)

// args for setting a client's password
// used in http and rrn messaging
type AdminSetPasswordArgs struct {
	UserName string `json:"username"`
	Password string `json:"password"`
}

// args for setting a client's role
// used in rrn messaging
type AdminSetRoleArgs struct {
	ClientID string `json:"clientID"`
	Role     string `json:"role"`
}

// args for updating a client's profile
// used in http and rrn messaging
type AdminUpdateProfileArgs struct {
	ClientID string              `json:"clientID"`
	Profile  authn.ClientProfile `json:"profile"`
}

// AdminAddClientArgs defines the arguments of the addAgent function
// Add Agent - Create an account for IoT device agents
type AdminAddClientArgs struct {

	// ClientID with Client ID
	ClientID string `json:"clientID,omitempty"`

	// DisplayName with Display Name
	DisplayName string `json:"displayName,omitempty"`

	// Optional Client password
	Password string `json:"password,omitempty`

	// PubKey with Public Key
	PubKey string `json:"pubKey,omitempty"`

	// Role of the client
	Role authn.ClientRole `json:"role,omitempty"`
}

// AdminGetProfilesResp defines the response of the getProfiles function
// Get Profiles - Get a list of all client profiles
// AdminGetProfilesResp defines a Client Profiles data schema.
type AdminGetProfilesResp []struct {

	// AdminGetProfilesResp with Client Profile
	AdminGetProfilesResp *authn.ClientProfile `json:"AdminGetProfilesResp,omitempty"`
}

// Handle the admin RRN request
func HandleAuthnAdminRequest(m authn.IAuthnModule, req *msg.RequestMessage, replyTo msg.ResponseHandler) error {
	var output any
	var err error
	switch req.Name {

	case AdminActionAddClient:
		args := AdminAddClientArgs{}
		err = utils.DecodeAsObject(req.Input, &args)
		if err == nil {
			err = m.AddClient(args.ClientID, args.DisplayName, args.Role, args.PubKey)
		}
	case AdminActionGetProfile:
		var clientID string
		err = utils.DecodeAsObject(req.Input, &clientID)
		if err == nil {
			output, err = m.GetProfile(clientID)
		}
	case AdminActionGetProfiles:
		if err == nil {
			output, err = m.GetProfiles()
		}
	case AdminActionRemoveClient:
		var clientID string
		err = utils.DecodeAsObject(req.Input, &clientID)
		if err == nil {
			err = m.RemoveClient(clientID)
		}
	case AdminActionSetPassword:
		var args AdminSetPasswordArgs // same as user
		err = utils.DecodeAsObject(req.Input, &args)
		if err == nil {
			err = m.SetPassword(args.UserName, args.Password)
		}
	case AdminActionUpdateProfile:
		var profile authn.ClientProfile
		err = utils.DecodeAsObject(req.Input, &profile)
		if err == nil {
			err = m.UpdateProfile(req.SenderID, profile)
		}
	default:
		err = errors.New("Unknown Method '" + req.Name + "' of service '" + req.ThingID + "'")
	}
	resp := req.CreateResponse(output, err)
	replyTo(resp)
	return nil
}

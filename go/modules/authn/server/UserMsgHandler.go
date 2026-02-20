package server

import (
	_ "embed"
	"errors"

	"github.com/hiveot/hivekit/go/modules/authn"
	"github.com/hiveot/hivekit/go/msg"
	"github.com/hiveot/hivekit/go/utils"
)

// Embed the user service TM
//
//go:embed "authn-user-tm.json"
var AuthnUserTMJson []byte

// AuthnUserThingID is the Thing instance ID of the user facing auth service.
const AuthnUserServiceID = "AuthnUser"

const (
	// Property names

	// Event names

	// Action names
	UserActionGetProfile    = "getProfile"
	UserActionLogout        = "Logout"
	UserActionRefreshToken  = "refreshToken"
	UserActionSetPassword   = "setPassword"
	UserActionUpdateProfile = "updateProfile"
)

// UserSetPasswordArgs defines the arguments of the setClientPassword function
// Set Client Password - Update the password of a consumer
//
// Client ID and password
type UserSetPasswordArgs struct {

	// ClientID with Client ID
	ClientID string `json:"clientID,omitempty"`

	// Password with Password
	Password string `json:"password,omitempty"`
}

// UserMsgServer provides the RRN messaging server for the authn user service.
type UserMsgHandler struct {
	m authn.IAuthnModule
}

// HandleAuthnUserRequest returns the RRN handler for the auth user requests.
func HandleAuthnUserRequest(m authn.IAuthnModule, req *msg.RequestMessage, replyTo msg.ResponseHandler) error {
	var output any
	var err error
	switch req.Name {

	case UserActionGetProfile:
		if err == nil {
			output, err = m.GetProfile(req.SenderID)
		} else {
			err = errors.New("bad function argument: " + err.Error())
		}

	case UserActionLogout:
		m.Logout(req.SenderID)

	case UserActionRefreshToken:
		var oldToken string
		err = utils.DecodeAsObject(req.Input, &oldToken)
		if err == nil {
			output, _, err = m.RefreshToken(req.SenderID, oldToken)
		} else {
			err = errors.New("bad function argument: " + err.Error())
		}

	case UserActionSetPassword:
		var newPassword string
		err = utils.DecodeAsObject(req.Input, &newPassword)
		if err == nil {
			err = m.SetPassword(req.SenderID, newPassword)
		} else {
			err = errors.New("bad function argument: " + err.Error())
		}

	case UserActionUpdateProfile:
		var profile authn.ClientProfile

		err = utils.DecodeAsObject(req.Input, &profile)
		if err == nil {
			err = m.UpdateProfile(req.SenderID, profile)
		} else {
			err = errors.New("bad function argument: " + err.Error())
		}

	default:
		err = errors.New("Unknown action '" + req.Name + "' of service '" + req.ThingID + "'")
	}
	resp := req.CreateResponse(output, err)
	err = replyTo(resp)
	return err
}

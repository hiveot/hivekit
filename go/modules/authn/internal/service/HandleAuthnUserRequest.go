package service

import (
	_ "embed"
	"errors"

	authnapi "github.com/hiveot/hivekit/go/modules/authn/api"
	"github.com/hiveot/hivekit/go/msg"
	"github.com/hiveot/hivekit/go/utils"
)

// Embed the user service TM
//
//go:embed "authn-user-tm.json"
var AuthnUserTMJson []byte

// HandleAuthnUserRequest returns the RRN handler for the auth user requests.
func HandleAuthnUserRequest(m authnapi.IAuthnService, req *msg.RequestMessage, replyTo msg.ResponseHandler) error {
	var output any
	var err error
	switch req.Name {

	case authnapi.UserActionGetProfile:
		if err == nil {
			output, err = m.GetProfile(req.SenderID)
		} else {
			err = errors.New("bad function argument: " + err.Error())
		}

	case authnapi.UserActionLogout:
		m.Logout(req.SenderID)

	case authnapi.UserActionRefreshToken:
		var oldToken string
		err = utils.DecodeAsObject(req.Input, &oldToken)
		if err == nil {
			output, _, err = m.RefreshToken(req.SenderID, oldToken)
		} else {
			err = errors.New("bad function argument: " + err.Error())
		}

	case authnapi.UserActionSetPassword:
		var newPassword string
		err = utils.DecodeAsObject(req.Input, &newPassword)
		if err == nil {
			err = m.SetPassword(req.SenderID, newPassword)
		} else {
			err = errors.New("bad function argument: " + err.Error())
		}

	case authnapi.UserActionUpdateProfile:
		var profile authnapi.ClientProfile

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

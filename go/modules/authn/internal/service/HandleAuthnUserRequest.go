package service

import (
	_ "embed"
	"errors"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/modules/authn"
	"github.com/hiveot/hivekit/go/utils"
)

// HandleAuthnUserRequest returns the RRN handler for the auth user requests.
func HandleAuthnUserRequest(m authn.IAuthnService, req *msg.RequestMessage, replyTo msg.ResponseHandler) error {
	var output any
	var err error
	switch req.Name {

	case authn.UserActionGetProfile:
		if err == nil {
			output, err = m.GetProfile(req.SenderID)
		} else {
			err = errors.New("bad function argument: " + err.Error())
		}

	case authn.UserActionLogout:
		aa := m.GetSessionManager()
		aa.Logout(req.SenderID)

	case authn.UserActionRefreshToken:
		var oldToken string
		err = utils.DecodeAsObject(req.Input, &oldToken)
		if err == nil {
			aa := m.GetSessionManager()
			output, _, err = aa.RefreshToken(req.SenderID, oldToken)
		} else {
			err = errors.New("bad function argument: " + err.Error())
		}

	case authn.UserActionSetPassword:
		var newPassword string
		err = utils.DecodeAsObject(req.Input, &newPassword)
		if err == nil {
			err = m.SetPassword(req.SenderID, newPassword)
		} else {
			err = errors.New("bad function argument: " + err.Error())
		}

	case authn.UserActionUpdateProfile:
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

package service

import (
	_ "embed"
	"errors"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/modules/authn"
	"github.com/hiveot/hivekit/go/utils"
)

// Handle the admin RRN request
func HandleAuthnAdminRequest(m authn.IAuthnService, req *msg.RequestMessage, replyTo msg.ResponseHandler) error {
	var output any
	var err error
	switch req.Name {

	case authn.AdminActionAddClient:
		args := authn.AdminAddClientArgs{}
		err = utils.DecodeAsObject(req.Input, &args)
		if err == nil {
			err = m.AddClient(args.ClientID, args.DisplayName, args.Role)
		}
	case authn.AdminActionGetProfile:
		var clientID string
		err = utils.DecodeAsObject(req.Input, &clientID)
		if err == nil {
			output, err = m.GetProfile(clientID)
		}
	case authn.AdminActionGetProfiles:
		output, err = m.GetProfiles()
	case authn.AdminActionRemoveClient:
		var clientID string
		err = utils.DecodeAsObject(req.Input, &clientID)
		if err == nil {
			err = m.RemoveClient(clientID)
		}
	case authn.AdminActionSetPassword:
		var args authn.AdminSetPasswordArgs // same as user
		err = utils.DecodeAsObject(req.Input, &args)
		if err == nil {
			err = m.SetPassword(args.UserName, args.Password)
		}
	case authn.AdminActionUpdateProfile:
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

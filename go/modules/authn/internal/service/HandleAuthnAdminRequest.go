package service

import (
	_ "embed"
	"errors"

	authnapi "github.com/hiveot/hivekit/go/modules/authn/api"
	"github.com/hiveot/hivekit/go/msg"
	"github.com/hiveot/hivekit/go/utils"
)

// Embed the admin service TM
//
//go:embed "authn-admin-tm.json"
var AuthnAdminTMJson []byte

// Handle the admin RRN request
func HandleAuthnAdminRequest(m authnapi.IAuthnService, req *msg.RequestMessage, replyTo msg.ResponseHandler) error {
	var output any
	var err error
	switch req.Name {

	case authnapi.AdminActionAddClient:
		args := authnapi.AdminAddClientArgs{}
		err = utils.DecodeAsObject(req.Input, &args)
		if err == nil {
			err = m.AddClient(args.ClientID, args.DisplayName, args.Role)
		}
	case authnapi.AdminActionGetProfile:
		var clientID string
		err = utils.DecodeAsObject(req.Input, &clientID)
		if err == nil {
			output, err = m.GetProfile(clientID)
		}
	case authnapi.AdminActionGetProfiles:
		output, err = m.GetProfiles()
	case authnapi.AdminActionRemoveClient:
		var clientID string
		err = utils.DecodeAsObject(req.Input, &clientID)
		if err == nil {
			err = m.RemoveClient(clientID)
		}
	case authnapi.AdminActionSetPassword:
		var args authnapi.AdminSetPasswordArgs // same as user
		err = utils.DecodeAsObject(req.Input, &args)
		if err == nil {
			err = m.SetPassword(args.UserName, args.Password)
		}
	case authnapi.AdminActionUpdateProfile:
		var profile authnapi.ClientProfile
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

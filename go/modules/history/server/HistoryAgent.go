package server

import (
	"fmt"

	"github.com/hiveot/hivekit/go/modules/history"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/msg"
	"github.com/hiveot/hivekit/go/wot"
)

// StartMsgHandler returns a new instance of the messaging handler for the history services.
// This uses the given connected transport for publishing events and subscribing to actions.
// The transport must be closed by the caller after use.
// If the transport is nil then use the HandleMessage method directly to pass methods to the agent,
// for example when testing.
//
//	svc is the history service whose capabilities to expose
//	ag is the optional connected agent connected to the server protocol
func StartMsgHandler(svc history.IHistoryModule) {

	// TODO: load latest retention rules from state store
	manageHistoryMethods := map[string]interface{}{
		history.GetRetentionRuleMethod:  svc.manageHistSvc.GetRetentionRule,
		history.GetRetentionRulesMethod: svc.manageHistSvc.GetRetentionRules,
		history.SetRetentionRulesMethod: svc.manageHistSvc.SetRetentionRules,
	}
	readHistoryMethods := map[string]interface{}{
		history.CursorFirstMethod:   svc.readHistSvc.First,
		history.CursorLastMethod:    svc.readHistSvc.Last,
		history.CursorNextMethod:    svc.readHistSvc.Next,
		history.CursorNextNMethod:   svc.readHistSvc.NextN,
		history.CursorPrevMethod:    svc.readHistSvc.Prev,
		history.CursorPrevNMethod:   svc.readHistSvc.PrevN,
		history.CursorReleaseMethod: svc.readHistSvc.Release,
		history.CursorSeekMethod:    svc.readHistSvc.Seek,
		history.GetCursorMethod:     svc.readHistSvc.GetCursor,
		history.ReadHistoryMethod:   svc.readHistSvc.ReadHistory,
	}
	rah := hubagent.NewAgentHandler(history.ReadHistoryServiceID, readHistoryMethods)
	mah := hubagent.NewAgentHandler(history.ManageHistoryServiceID, manageHistoryMethods)

	// receive subscribed updates for events and properties
	ag.Consumer.SetNotificationHandler(func(notif *msg.NotificationMessage) {
		if notif.Operation == wot.OpSubscribeEvent {
			_ = svc.addHistory.AddMessage(notif)
		} else if notif.Operation == wot.OpObserveProperty {
			_ = svc.addHistory.AddMessage(notif)
		}
		//ignore the rest
		return
	})

	// handle service requests
	ag.SetRequestHandler(func(req *msg.RequestMessage, c transports.IConnection) *msg.ResponseMessage {
		if req.Operation == vocab.OpInvokeAction {
			if req.ThingID == history.ReadHistoryServiceID {
				return rah.HandleRequest(req, c)
			} else if req.ThingID == history.ManageHistoryServiceID {
				return mah.HandleRequest(req, c)
			}
		}
		return req.CreateResponse(nil, fmt.Errorf("Unhandled message"))
	})

	// TODO: publish the TD
}

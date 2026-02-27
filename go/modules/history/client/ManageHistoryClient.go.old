package historyclient

import (
	"github.com/hiveot/hivekit/go/modules/clients"
	"github.com/hiveot/hivekit/go/modules/history"
	"github.com/hiveot/hivekit/go/wot/td"
)

// ManageHistoryClient client for managing retention of the history service
type ManageHistoryClient struct {
	// service providing the history management capability
	dThingID string
	//co       transports.IClientConnection
	co *clients.Consumer
}

// GetRetentionRule returns the retention configuration of an event by name
// This applies to events from any publishers and things
// returns nil if there is no retention rule for the event
//
//	dThingID
//	eventName whose retention to return
func (cl *ManageHistoryClient) GetRetentionRule(
	dThingID string, name string) (*history.RetentionRule, error) {

	args := history.GetRetentionRuleArgs{
		ThingID: dThingID,
		Name:    name,
	}
	resp := history.GetRetentionRuleResp{}
	err := cl.co.InvokeAction(cl.dThingID, history.GetRetentionRuleMethod, &args, &resp)
	return resp.Rule, err
}

// GetRetentionRules returns the list of retention rules
func (cl *ManageHistoryClient) GetRetentionRules() (history.RetentionRuleSet, error) {
	resp := history.GetRetentionRulesResp{}
	err := cl.co.InvokeAction(cl.dThingID, history.GetRetentionRulesMethod, nil, &resp)
	return resp.Rules, err
}

// SetRetentionRules configures the retention of a Thing event
func (cl *ManageHistoryClient) SetRetentionRules(rules history.RetentionRuleSet) error {
	args := history.SetRetentionRulesArgs{Rules: rules}
	err := cl.co.InvokeAction(cl.dThingID, history.SetRetentionRulesMethod, &args, nil)
	return err
}

// NewManageHistoryClient creates a new instance of the manage history client for use by authorized clients
func NewManageHistoryClient(co *clients.Consumer) *ManageHistoryClient {
	agentID := history.AgentID
	mngCl := &ManageHistoryClient{
		dThingID: td.MakeDigiTwinThingID(agentID, history.ManageHistoryServiceID),
		co:       co,
	}
	return mngCl
}

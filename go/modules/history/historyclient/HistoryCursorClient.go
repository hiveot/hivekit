package historyclient

import "C"
import (
	"time"

	"github.com/hiveot/hivekit/go/modules/clients"
	"github.com/hiveot/hivekit/go/msg"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hivekit/go/wot"
	"github.com/hiveot/hivekit/go/wot/td"
)

// HistoryCursorClient provides iterator client for iterating the history
type HistoryCursorClient struct {
	// the key identifying this cursor
	cursorKey string

	// history cursor service ID
	dThingID string
	co       *clients.Consumer
}

// First positions the cursor at the first key in the ordered list
// This returns an error if the cursor has expired or is not found.
func (cl *HistoryCursorClient) First() (thingValue *msg.ThingValue, valid bool, err error) {
	req := history.CursorArgs{
		CursorKey: cl.cursorKey,
	}
	resp := history.CursorSingleResp{}
	err = cl.co.Rpc(wot.OpInvokeAction, cl.dThingID, history.CursorFirstMethod, &req, &resp)
	//err = cl.co.SendRequest(cl.dThingID, server.CursorFirstMethod, &req, &resp)
	return resp.Value, resp.Valid, err
}

// Last positions the cursor at the last key in the ordered list
// This returns an error if the cursor has expired or is not found.
func (cl *HistoryCursorClient) Last() (
	thingValue *msg.ThingValue, valid bool, err error) {

	req := history.CursorArgs{
		CursorKey: cl.cursorKey,
	}
	resp := history.CursorSingleResp{}
	err = cl.co.Rpc(wot.OpInvokeAction, cl.dThingID, history.CursorLastMethod, &req, &resp)
	return resp.Value, resp.Valid, err
}

// Next moves the cursor to the next key from the current cursor
// This returns an error if the cursor has expired or is not found.
func (cl *HistoryCursorClient) Next() (
	thingValue *msg.ThingValue, valid bool, err error) {

	req := history.CursorArgs{
		CursorKey: cl.cursorKey,
	}
	resp := history.CursorSingleResp{}
	err = cl.co.Rpc(wot.OpInvokeAction, cl.dThingID, history.CursorNextMethod, &req, &resp)
	return resp.Value, resp.Valid, err
}

// NextN moves the cursor to the next N steps from the current cursor
// This returns an error if the cursor has expired or is not found.
func (cl *HistoryCursorClient) NextN(
	limit int, until string) (batch []*msg.ThingValue, itemsRemaining bool, err error) {

	req := history.CursorNArgs{
		CursorKey: cl.cursorKey,
		Until:     until,
		Limit:     limit,
	}
	resp := history.CursorNResp{}
	err = cl.co.Rpc(wot.OpInvokeAction, cl.dThingID, history.CursorNextNMethod, &req, &resp)
	return resp.Values, resp.ItemsRemaining, err
}

// Prev moves the cursor to the previous key from the current cursor
// This returns an error if the cursor has expired or is not found.
func (cl *HistoryCursorClient) Prev() (
	thingValue *msg.ThingValue, valid bool, err error) {

	req := history.CursorArgs{
		CursorKey: cl.cursorKey,
	}
	resp := history.CursorSingleResp{}
	err = cl.co.Rpc(wot.OpInvokeAction, cl.dThingID, history.CursorPrevMethod, &req, &resp)
	return resp.Value, resp.Valid, err
}

// PrevN moves the cursor to the previous N steps from the current cursor
// This returns an error if the cursor has expired or is not found.
func (cl *HistoryCursorClient) PrevN(
	limit int, until string) (batch []*msg.ThingValue, itemsRemaining bool, err error) {

	req := history.CursorNArgs{
		CursorKey: cl.cursorKey,
		Until:     until,
		Limit:     limit,
	}
	resp := history.CursorNResp{}
	err = cl.co.Rpc(wot.OpInvokeAction, cl.dThingID, history.CursorPrevNMethod, &req, &resp)
	return resp.Values, resp.ItemsRemaining, err
}

// Release the cursor capability
func (cl *HistoryCursorClient) Release() {
	req := history.CursorReleaseArgs{
		CursorKey: cl.cursorKey,
	}
	err := cl.co.Rpc(wot.OpInvokeAction, cl.dThingID, history.CursorReleaseMethod, &req, nil)
	_ = err
	return
}

// Seek the starting point for iterating the history
// timeStamp in ISO8106 format
// This returns an error if the cursor has expired or is not found.
func (cl *HistoryCursorClient) Seek(timeStamp time.Time) (
	thingValue *msg.ThingValue, valid bool, err error) {
	timeStampStr := utils.FormatUTCMilli(timeStamp)
	req := history.CursorSeekArgs{
		CursorKey: cl.cursorKey,
		TimeStamp: timeStampStr,
	}
	resp := history.CursorSingleResp{}
	err = cl.co.Rpc(wot.OpInvokeAction, cl.dThingID, history.CursorSeekMethod, &req, &resp)
	return resp.Value, resp.Valid, err
}

// NewHistoryCursorClient returns a read cursor client
// Intended for internal use.
//
//	co client connection to the Hub
//	serviceID of the read capability
//	cursorKey is the iterator key obtain when requesting the cursor
func NewHistoryCursorClient(co *clients.Consumer, cursorKey string) *HistoryCursorClient {
	agentID := history.AgentID
	serviceID := history.ReadHistoryServiceID
	cl := &HistoryCursorClient{
		cursorKey: cursorKey,
		// history cursor serviceID
		dThingID: td.MakeDigiTwinThingID(agentID, serviceID),
		co:       co,
	}
	return cl
}

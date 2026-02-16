package historyclient

import (
	"log/slog"
	"time"

	"github.com/hiveot/hivekit/go/modules/clients"
	historyserver "github.com/hiveot/hivekit/go/modules/history/server"
	"github.com/hiveot/hivekit/go/msg"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hivekit/go/wot"
)

// ReadHistoryClient for talking to the history service
// This client supports both the cursor-based iteration and the batch read history method.
//
// To use the cursor-based iteration, use GetCursor to obtain a cursor and then use the cursor
// methods to iterate through the history.
type ReadHistoryClient struct {
	// Agent that handles the ThingID requests
	histAgentID string
	// ThingID of the service providing the read history capability
	histThingID string
	// consumer instance to use for sending requests
	co *clients.Consumer
}

// GetCursor obtains a cursor key to iterate using the iteration functions.
// This returns a release function that MUST be called after use.
//
// Cursor keys expire after a short period of inactivity. Defaults to 1 minute.
//
//	thingID the event or action belongs to
//	filterOnName optiona; filter on a specific event or action name
func (cl *ReadHistoryClient) GetCursor(thingID string, filterOnName string) (
	cursorKey string, releaseFn func(), err error) {

	args := historyserver.CreateCursorArgs{
		ThingID: thingID,
		Name:    filterOnName,
	}
	err = cl.co.Rpc(wot.OpInvokeAction,
		cl.histThingID, historyserver.CreateCursorMethod, &args, &cursorKey)
	return cursorKey, func() { cl.ReleaseCursor(cursorKey) }, err
}

// First positions the cursor at the first key in the ordered list
// This returns an error if the cursor has expired or is not found.
func (cl *ReadHistoryClient) First(cursorKey string) (thingValue *msg.ThingValue, valid bool, err error) {
	resp := historyserver.CursorValueResp{}
	err = cl.co.Rpc(wot.OpInvokeAction,
		cl.histThingID, historyserver.CursorFirstMethod, &cursorKey, &resp)
	return resp.Value, resp.Valid, err
}

// Last positions the cursor at the last key in the ordered list
// This returns an error if the cursor has expired or is not found.
func (cl *ReadHistoryClient) Last(cursorKey string) (thingValue *msg.ThingValue, valid bool, err error) {
	resp := historyserver.CursorValueResp{}
	err = cl.co.Rpc(wot.OpInvokeAction,
		cl.histThingID, historyserver.CursorLastMethod, &cursorKey, &resp)
	return resp.Value, resp.Valid, err
}

// Next moves the cursor to the next key from the current cursor
// This returns an error if the cursor has expired or is not found.
func (cl *ReadHistoryClient) Next(cursorKey string) (thingValue *msg.ThingValue, valid bool, err error) {
	resp := historyserver.CursorValueResp{}
	err = cl.co.Rpc(wot.OpInvokeAction,
		cl.histThingID, historyserver.CursorNextMethod, &cursorKey, &resp)
	return resp.Value, resp.Valid, err
}

// NextN moves the cursor to the next N steps from the current cursor
// This returns an error if the cursor has expired or is not found.
func (cl *ReadHistoryClient) NextN(cursorKey string, until time.Time, limit int) (
	value []*msg.ThingValue, itemsRemaining bool, err error) {

	untilRFC := utils.FormatUTCMilli(until)
	req := historyserver.CursorNArgs{
		CursorKey: cursorKey,
		Until:     untilRFC,
		Limit:     limit,
	}
	resp := historyserver.CursorNResp{}
	err = cl.co.Rpc(wot.OpInvokeAction,
		cl.histThingID, historyserver.CursorNextNMethod, &req, &resp)
	return resp.Values, resp.ItemsRemaining, err
}

// Prev moves the cursor to the previous key from the current cursor
// This returns an error if the cursor has expired or is not found.
func (cl *ReadHistoryClient) Prev(cursorKey string) (thingValue *msg.ThingValue, valid bool, err error) {
	resp := historyserver.CursorValueResp{}
	err = cl.co.Rpc(wot.OpInvokeAction,
		cl.histThingID, historyserver.CursorPrevMethod, &cursorKey, &resp)
	return resp.Value, resp.Valid, err
}

// PrevN moves the cursor to the previous N steps from the current cursor and returns
// the batch of values and whether there are more items remaining.
// This returns an error if the cursor has expired or is not found.
func (cl *ReadHistoryClient) PrevN(cursorKey string, until time.Time, limit int) (
	value []*msg.ThingValue, itemsRemaining bool, err error) {

	untilRFC := utils.FormatUTCMilli(until)
	req := historyserver.CursorNArgs{
		CursorKey: cursorKey,
		Until:     untilRFC,
		Limit:     limit,
	}
	resp := historyserver.CursorNResp{}
	err = cl.co.Rpc(wot.OpInvokeAction,
		cl.histThingID, historyserver.CursorPrevNMethod, &req, &resp)
	return resp.Values, resp.ItemsRemaining, err
}

// Release the allocated cursor key
func (cl *ReadHistoryClient) ReleaseCursor(cursorKey string) {
	err := cl.co.Rpc(wot.OpInvokeAction,
		cl.histThingID, historyserver.CursorReleaseMethod, &cursorKey, nil)
	if err != nil {
		slog.Warn("ReleaseCursor failed", "cursorKey", cursorKey, "err", err)
	}
}

// ReadHistory returns a list of historical messages in time order.
//
//	thingID the event or action belongs to
//	FilterOnName option filter on a specific event or action name
//	timestamp to start/end
//	duration number of seconds to return. Use negative number to go back in time.
//	limit max nr of items to return. Use 0 for max limit
//
// This returns a list of messages and a flag indicating of all duration was returned
// or whether items were remaining. If items were remaining them use the last entry
// to continue reading the next page.
func (cl *ReadHistoryClient) ReadHistory(thingID string, filterOnName string,
	timestamp time.Time, duration time.Duration, limit int) (
	batch []*msg.ThingValue, itemsRemaining bool, err error) {

	args := historyserver.ReadHistoryArgs{
		ThingID:        thingID,
		AffordanceName: filterOnName,
		Timestamp:      timestamp.Format(time.RFC3339),
		Duration:       int(duration.Seconds()),
		Limit:          limit,
	}
	resp := historyserver.ReadHistoryResp{}
	err = cl.co.Rpc(wot.OpInvokeAction,
		cl.histThingID, historyserver.ReadHistoryMethod, &args, &resp)
	return resp.Values, resp.ItemsRemaining, err
}

// Seek the starting point for iterating the history
// This returns an error if the cursor has expired or is not found.
func (cl *ReadHistoryClient) Seek(cursorKey string, timestamp time.Time) (
	thingValue *msg.ThingValue, valid bool, err error) {
	timeRFC := utils.FormatUTCMilli(timestamp)
	args := historyserver.CursorSeekArgs{
		CursorKey: cursorKey,
		Timestamp: timeRFC,
	}
	resp := historyserver.CursorValueResp{}
	err = cl.co.Rpc(wot.OpInvokeAction,
		cl.histThingID, historyserver.CursorSeekMethod, &args, &resp)
	return resp.Value, resp.Valid, err
}

// NewReadHistoryClient returns an instance of the read history client using the given
// consumer connection.
//
//	invokeAction is the TD invokeAction for the invoke-action operation of the history service
func NewReadHistoryClient(co *clients.Consumer) *ReadHistoryClient {
	// how to determine the agentID and thingID of the history service?
	// For now we use the well-known IDs. In future this needs discovery
	agentID := historyserver.ReadHistoryServiceID
	histCl := ReadHistoryClient{
		co:          co,
		histAgentID: agentID,
		histThingID: historyserver.ReadHistoryServiceID,
	}
	return &histCl
}

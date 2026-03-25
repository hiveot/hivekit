package history

import (
	"log/slog"
	"time"

	"github.com/hiveot/hivekit/go/modules/clients"
	historyapi "github.com/hiveot/hivekit/go/modules/history/api"
	"github.com/hiveot/hivekit/go/msg"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hivekit/go/wot/td"
)

// ReadHistoryClient for talking to the history service
// This client supports both the cursor-based iteration and the batch read history method.
//
// To use the cursor-based iteration, use GetCursor to obtain a cursor and then use the cursor
// methods to iterate through the history.
type ReadHistoryClient struct {
	// Agent that handles the ThingID requests
	// histAgentID string
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

	args := historyapi.CreateCursorArgs{
		ThingID: thingID,
		Name:    filterOnName,
	}
	err = cl.co.Rpc(td.OpInvokeAction,
		cl.histThingID, historyapi.CreateCursorMethod, &args, &cursorKey)
	return cursorKey, func() { cl.ReleaseCursor(cursorKey) }, err
}

// First positions the cursor at the first key in the ordered list
// This returns an error if the cursor has expired or is not found.
func (cl *ReadHistoryClient) First(cursorKey string) (value *msg.NotificationMessage, valid bool, err error) {
	resp := historyapi.CursorValueResp{}
	err = cl.co.Rpc(td.OpInvokeAction,
		cl.histThingID, historyapi.CursorFirstMethod, &cursorKey, &resp)
	return resp.Value, resp.Valid, err
}

// Last positions the cursor at the last key in the ordered list
// This returns an error if the cursor has expired or is not found.
func (cl *ReadHistoryClient) Last(cursorKey string) (thingValue *msg.NotificationMessage, valid bool, err error) {
	resp := historyapi.CursorValueResp{}
	err = cl.co.Rpc(td.OpInvokeAction,
		cl.histThingID, historyapi.CursorLastMethod, &cursorKey, &resp)
	return resp.Value, resp.Valid, err
}

// Next moves the cursor to the next key from the current cursor
// This returns an error if the cursor has expired or is not found.
func (cl *ReadHistoryClient) Next(cursorKey string) (thingValue *msg.NotificationMessage, valid bool, err error) {
	resp := historyapi.CursorValueResp{}
	err = cl.co.Rpc(td.OpInvokeAction,
		cl.histThingID, historyapi.CursorNextMethod, &cursorKey, &resp)
	return resp.Value, resp.Valid, err
}

// NextN moves the cursor to the next N steps from the current cursor
// This returns an error if the cursor has expired or is not found.
func (cl *ReadHistoryClient) NextN(cursorKey string, until time.Time, limit int) (
	value []*msg.NotificationMessage, itemsRemaining bool, err error) {

	untilRFC := utils.FormatUTCMilli(until)
	req := historyapi.CursorNArgs{
		CursorKey: cursorKey,
		Until:     untilRFC,
		Limit:     limit,
	}
	resp := historyapi.CursorNResp{}
	err = cl.co.Rpc(td.OpInvokeAction,
		cl.histThingID, historyapi.CursorNextNMethod, &req, &resp)
	return resp.Values, resp.ItemsRemaining, err
}

// Prev moves the cursor to the previous key from the current cursor
// This returns an error if the cursor has expired or is not found.
func (cl *ReadHistoryClient) Prev(cursorKey string) (thingValue *msg.NotificationMessage, valid bool, err error) {
	resp := historyapi.CursorValueResp{}
	err = cl.co.Rpc(td.OpInvokeAction,
		cl.histThingID, historyapi.CursorPrevMethod, &cursorKey, &resp)
	return resp.Value, resp.Valid, err
}

// PrevN moves the cursor to the previous N steps from the current cursor and returns
// the batch of values and whether there are more items remaining.
// This returns an error if the cursor has expired or is not found.
func (cl *ReadHistoryClient) PrevN(cursorKey string, until time.Time, limit int) (
	value []*msg.NotificationMessage, itemsRemaining bool, err error) {

	untilRFC := utils.FormatUTCMilli(until)
	req := historyapi.CursorNArgs{
		CursorKey: cursorKey,
		Until:     untilRFC,
		Limit:     limit,
	}
	resp := historyapi.CursorNResp{}
	err = cl.co.Rpc(td.OpInvokeAction,
		cl.histThingID, historyapi.CursorPrevNMethod, &req, &resp)
	return resp.Values, resp.ItemsRemaining, err
}

// Release the allocated cursor key
func (cl *ReadHistoryClient) ReleaseCursor(cursorKey string) {
	err := cl.co.Rpc(td.OpInvokeAction,
		cl.histThingID, historyapi.CursorReleaseMethod, &cursorKey, nil)
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
	batch []*msg.NotificationMessage, itemsRemaining bool, err error) {

	args := historyapi.ReadHistoryArgs{
		ThingID:        thingID,
		AffordanceName: filterOnName,
		Timestamp:      timestamp.Format(time.RFC3339),
		Duration:       int(duration.Seconds()),
		Limit:          limit,
	}
	resp := historyapi.ReadHistoryResp{}
	err = cl.co.Rpc(td.OpInvokeAction,
		cl.histThingID, historyapi.ReadHistoryMethod, &args, &resp)
	return resp.Values, resp.ItemsRemaining, err
}

// Seek the starting point for iterating the history
// This returns an error if the cursor has expired or is not found.
func (cl *ReadHistoryClient) Seek(cursorKey string, timestamp time.Time) (
	thingValue *msg.NotificationMessage, valid bool, err error) {

	timeRFC := utils.FormatUTCMilli(timestamp)
	args := historyapi.CursorSeekArgs{
		CursorKey: cursorKey,
		Timestamp: timeRFC,
	}
	resp := historyapi.CursorValueResp{}
	err = cl.co.Rpc(td.OpInvokeAction,
		cl.histThingID, historyapi.CursorSeekMethod, &args, &resp)
	return resp.Value, resp.Valid, err
}

// NewReadHistoryClient returns an instance of the read history client using the given
// consumer connection.
//
//	invokeAction is the TD invokeAction for the invoke-action operation of the history service
func NewReadHistoryClient(co *clients.Consumer) *ReadHistoryClient {
	// how to determine the thingID of the history service?
	// For now we use the well-known IDs. In future this needs discovery
	histCl := ReadHistoryClient{
		co:          co,
		histThingID: historyapi.DefaultHistoryModuleID,
	}
	return &histCl
}

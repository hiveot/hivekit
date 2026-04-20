package historypkg

import (
	"log/slog"
	"time"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/factory"
	"github.com/hiveot/hivekit/go/modules/history"
	"github.com/hiveot/hivekit/go/utils"
)

// ReadHistoryClient module for messaging the history service
// This client supports both the cursor-based iteration and the batch read history method.
//
// To use the cursor-based iteration, use GetCursor to obtain a cursor and then use the cursor
// methods to iterate through the history.
//
// This client module only handles the messaging. It must be linked to a transport client
// that connects to the service.
type ReadHistoryClient struct {
	modules.HiveModuleBase

	// ThingID of the service providing the read history capability
	histThingID string
}

// invoke action on the history service
func (cl *ReadHistoryClient) call(name string, input any, output any) error {
	return cl.Rpc("ReadHistoryClient", td.OpInvokeAction, cl.histThingID, name, input, output)
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

	args := history.CreateCursorArgs{
		ThingID: thingID,
		Name:    filterOnName,
	}
	err = cl.call(history.CreateCursorMethod, &args, &cursorKey)
	return cursorKey, func() { cl.ReleaseCursor(cursorKey) }, err
}

// First positions the cursor at the first key in the ordered list
// This returns an error if the cursor has expired or is not found.
func (cl *ReadHistoryClient) First(cursorKey string) (value *msg.NotificationMessage, valid bool, err error) {
	resp := history.CursorValueResp{}
	err = cl.call(history.CursorFirstMethod, &cursorKey, &resp)
	return resp.Value, resp.Valid, err
}

// Last positions the cursor at the last key in the ordered list
// This returns an error if the cursor has expired or is not found.
func (cl *ReadHistoryClient) Last(cursorKey string) (thingValue *msg.NotificationMessage, valid bool, err error) {
	resp := history.CursorValueResp{}
	err = cl.call(history.CursorLastMethod, &cursorKey, &resp)
	return resp.Value, resp.Valid, err
}

// Next moves the cursor to the next key from the current cursor
// This returns an error if the cursor has expired or is not found.
func (cl *ReadHistoryClient) Next(cursorKey string) (thingValue *msg.NotificationMessage, valid bool, err error) {
	resp := history.CursorValueResp{}
	err = cl.call(history.CursorNextMethod, &cursorKey, &resp)
	return resp.Value, resp.Valid, err
}

// NextN moves the cursor to the next N steps from the current cursor
// This returns an error if the cursor has expired or is not found.
func (cl *ReadHistoryClient) NextN(cursorKey string, until time.Time, limit int) (
	value []*msg.NotificationMessage, itemsRemaining bool, err error) {

	untilRFC := utils.FormatUTCMilli(until)
	req := history.CursorNArgs{
		CursorKey: cursorKey,
		Until:     untilRFC,
		Limit:     limit,
	}
	resp := history.CursorNResp{}
	err = cl.call(history.CursorNextNMethod, &req, &resp)
	return resp.Values, resp.ItemsRemaining, err
}

// Prev moves the cursor to the previous key from the current cursor
// This returns an error if the cursor has expired or is not found.
func (cl *ReadHistoryClient) Prev(cursorKey string) (thingValue *msg.NotificationMessage, valid bool, err error) {
	resp := history.CursorValueResp{}
	err = cl.call(history.CursorPrevMethod, &cursorKey, &resp)
	return resp.Value, resp.Valid, err
}

// PrevN moves the cursor to the previous N steps from the current cursor and returns
// the batch of values and whether there are more items remaining.
// This returns an error if the cursor has expired or is not found.
func (cl *ReadHistoryClient) PrevN(cursorKey string, until time.Time, limit int) (
	value []*msg.NotificationMessage, itemsRemaining bool, err error) {

	untilRFC := utils.FormatUTCMilli(until)
	req := history.CursorNArgs{
		CursorKey: cursorKey,
		Until:     untilRFC,
		Limit:     limit,
	}
	resp := history.CursorNResp{}
	err = cl.call(history.CursorPrevNMethod, &req, &resp)
	return resp.Values, resp.ItemsRemaining, err
}

// Release the allocated cursor key
func (cl *ReadHistoryClient) ReleaseCursor(cursorKey string) {
	err := cl.call(history.CursorReleaseMethod, &cursorKey, nil)
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

	args := history.ReadHistoryArgs{
		ThingID:        thingID,
		AffordanceName: filterOnName,
		Timestamp:      timestamp.Format(time.RFC3339),
		Duration:       int(duration.Seconds()),
		Limit:          limit,
	}
	resp := history.ReadHistoryResp{}
	err = cl.call(history.ReadHistoryMethod, &args, &resp)
	return resp.Values, resp.ItemsRemaining, err
}

// Seek the starting point for iterating the history
// This returns an error if the cursor has expired or is not found.
func (cl *ReadHistoryClient) Seek(cursorKey string, timestamp time.Time) (
	thingValue *msg.NotificationMessage, valid bool, err error) {

	timeRFC := utils.FormatUTCMilli(timestamp)
	args := history.CursorSeekArgs{
		CursorKey: cursorKey,
		Timestamp: timeRFC,
	}
	resp := history.CursorValueResp{}
	err = cl.call(history.CursorSeekMethod, &args, &resp)
	return resp.Value, resp.Valid, err
}

// NewReadHistoryClient returns an instance of the read history client.
//
// The client must be linked to a client connection.
// (multiple clients can be chained this way)
//
//	invokeAction is the TD invokeAction for the invoke-action operation of the history service
func NewReadHistoryClient() *ReadHistoryClient {
	// how to determine the thingID of the history service?
	// For now we use the well-known IDs. In future this needs discovery
	histCl := ReadHistoryClient{
		histThingID: history.HistoryModuleType,
	}
	return &histCl
}

// NewReadHistoryClientFactory returns an instance of the read history client
// using the given factory environment.
//
// The client must be linked to a client connection.
// (multiple clients can be chained this way)
//
//	invokeAction is the TD invokeAction for the invoke-action operation of the history service
func NewReadHistoryClientFactory(f factory.IModuleFactory) modules.IHiveModule {
	return NewReadHistoryClient()
}

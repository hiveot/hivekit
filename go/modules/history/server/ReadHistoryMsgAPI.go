package historyserver

import (
	"github.com/hiveot/hivekit/go/msg"
)

// ReadHistoryServiceID is the thingID of the reading service
const ReadHistoryServiceID = "historyReader"

// Action names as defined in the module TM
const (

	// CreateCursorMethod returns a cursor to iterate the history of a things
	// The cursor MUST be released after use.
	// The cursor will expire after not being used for the default expiry period.
	//
	// Input: CreateCursorArgs with ThingID and Affordance name
	// Output: string with cursorKey
	CreateCursorMethod = "createCursor"

	// CursorFirstMethod return the oldest value in the history
	// Input: string with cursor key
	// Output: CursorValueResp
	CursorFirstMethod = "cursorFirst"

	// CursorLastMethod return the newest value in the history
	// Input: string with cursor key
	// Output:CursorValueResp
	CursorLastMethod = "cursorLast"

	// CursorNextMethod returns the next value in the history
	// Input: string with cursor key
	// Output:CursorValueResp
	CursorNextMethod = "cursorNext"

	// CursorNextNMethod returns a batch of next N historical values
	// Input: CursorNArgs
	// Output: CursorNResp
	CursorNextNMethod = "cursorNextN"

	// CursorPrevMethod returns the previous value in the history
	// Output: CursorValueResp
	CursorPrevMethod = "cursorPrev"

	// CursorPrevNMethod returns a batch of prior N historical values
	// Input: CursorNArgs
	// Output: CursorNResp
	CursorPrevNMethod = "cursorPrevN"

	// CursorReleaseMethod releases the cursor and resources
	// This MUST be called after the cursor is not longer used.
	// Input: string with cursorKey
	// Output: n/a
	CursorReleaseMethod = "cursorRelease"

	// CursorSeekMethod seeks the starting point in time for iterating the history
	// This returns a single value response with the value at timestamp or next closest
	// if it doesn't exist.
	// Returns empty value when there are no values at or past the given timestamp
	//
	// Input: CursorSeekArgs
	// Output: CursorValueResp
	CursorSeekMethod = "cursorSeek"

	// ReadHistoryMethod reads the history up to a limit.
	ReadHistoryMethod = "readHistory"
)

// CreateCursorArgs contain the thingID and affordance filter name for the cursor
type CreateCursorArgs struct {
	ThingID string `json:"thingID"`
	// Name is the optional affordance name to filter on, eg propName, eventName or actionName
	Name string `json:"name,omitempty"`
}

// CursorValueResp contains a single response value and valid status
// Used as output for First, Last, Next, Prev methods
type CursorValueResp struct {
	// The value at the new cursor position or nil if not valid
	Value *msg.ThingValue `json:"value"`
	// The current position holds a valid value
	Valid bool `json:"valid"`
}

// CursorNArgs contains the request for use in NextN and PrevN
type CursorNArgs struct {
	// Cursor identifier obtained with CreateCursor
	CursorKey string `json:"cursorKey"`
	// Maximum number of results to return
	Limit int `json:"limit,omitempty"`
	// Time until to keep reading or "" for up to 1 year
	Until string `json:"until,omitempty"`
}

// CursorNResp contains the batch response to NextN and PrevN
type CursorNResp struct {
	// Returns up to 'Limit' iterated values.
	// This will be an empty list when trying to read past the last value.
	Values []*msg.ThingValue `json:"values"`
	// There are still items remaining.
	ItemsRemaining bool `json:"itemsRemaining"`
}

type CursorSeekArgs struct {
	// Cursor identifier obtained with CreateCursor
	CursorKey string `json:"cursorKey"`
	// timestamp in rfc8601 format
	Timestamp string `json:"timeStamp"`
}

// ReadHistoryArgs arguments for reading a batch of historical values
type ReadHistoryArgs struct {
	// Thing to read values from
	ThingID string `json:"thingID"`
	// Optional filter value to search for a specific event,property or action name
	AffordanceName string `json:"name,omitempty"`
	// Timestamp in RFC3339 format or 'now' for default
	Timestamp string `json:"timeStamp"`
	// Duration to read or 0 for previous 24 hours (-3600*24)
	Duration int `json:"duration"`
	// limit nr of results or 0 for default of 1000
	Limit int `json:"limit"`
}

// ReadHistoryResp contains the batch response to a reading history values
type ReadHistoryResp struct {
	// Returns up to 'Limit' iterated values.
	// This will be an empty list when trying to read past the last value.
	Values []*msg.ThingValue `json:"values"`
	// There are still items remaining.
	ItemsRemaining bool `json:"itemsRemaining"`
}

package history

import (
	"time"

	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/msg"
)

// DefaultHistoryModuleID is the default moduleID of the history module.
const DefaultHistoryModuleID = "history"

// DefaultLimit nr items of none provided
const DefaultLimit = 1000

// HistoryConfig defines the configuration for the history module.
type HistoryConfig struct {
	// optional filter for notifications to retain. If not provided no notifications are retained.
	NotificationFilter *msg.MessageFilter `yaml:"notifications,omitempty"`
	// optional filter for requests to retain. If not provided no requests are retained.
	RequestFilter *msg.MessageFilter `yaml:"requests,omitempty"`
}

// IHistoryModule defines the interface to the directory service module
// This is implemented in the module and the client api
//
// The history persists values stored per ThingID. Values are ordered by timestamp and
// the affordance they belong to. The affordance the name used in the property, event and action
// affordance.
//
// The cursor functions use a so-called cursorKey to identify the cursor that is used
// in iterations. This approach enables the use of iteration cursors by a remote client
// that doesn't have access to a cursor object itself. To enable this, the server uses
// a cursor cache that stores cursors and expires them on release or after a timeout.
//
// To prevent cursor hijacking, it is linked to the authenticated clientID of the caller
// The caller's clientID of all iteration requests must match that of the cursor creator.
type IHistoryModule interface {
	modules.IHiveModule

	// CreateCursor creates a new iterator for reading historical values of a Thing.
	//
	// Cursors should be released using ReleaseCursor after usage. Unused cursors have
	// are automatically discarded after the configured time limit (default 1 minute).
	//
	//	clientID identifies the owner of the cursor
	//	thingID is the Thing whose data to iteration
	//	affName is the optional affordance name to iterate, or "" for any
	CreateCursor(clientID string, thingID string, affName string) (cursorKey string, err error)

	// First returns the oldest value in the history
	//
	// If an affordance name is provided it forwards to the first value for that affordance.
	//
	//	clientID must match the owner of the cursor
	//	cursorKey is the cursor to iterate.
	First(clientID string, cursorKey string) (
		value *msg.ThingValue, valid bool, err error)

	// Last positions the cursor at the last key in the ordered list
	// If an affordance name is provided then it rewinds to the first available value
	// for that affordance.
	//	clientID must match the owner of the cursor
	//	cursorKey is the cursor to iterate.
	Last(clientID string, cursorKey string) (
		tv *msg.ThingValue, valid bool, err error)

	// Next moves the cursor to the next key from the current cursor.
	// If affName was provided then continue iterating until the affordance name matches.
	// First() or Seek must have been called first.
	// This returns an error if the cursor is not valid.
	//	clientID must match the owner of the cursor
	//	cursorKey is the cursor to iterate.
	Next(clientID string, cursorKey string) (
		tv *msg.ThingValue, valid bool, err error)

	// NextN moves the cursor to the next N places from the current cursor and return a
	// list with N values in incremental time order.
	//
	// This returns the list with values and itemsRemaining, which is false if the iterator
	// has reached the end. Intended to speed up with batch iterations over rpc.
	//
	//	clientID must match the owner of the cursor
	//	cursorKey is the cursor to iterate.
	//	until is the time to iterate towards
	//	limit is the maximum number of items to return. 0 defaults to DefaultLimit.
	//
	// This returns a list of values in time order and 'itemsRemaining' which is
	// true if the iterator has reached the first match.
	NextN(clientID string, cursorKey string, until time.Time, limit int) (
		tvList []*msg.ThingValue, itemsRemaining bool, err error)

	// Prev moves the cursor to the previous key from the current cursor
	// Last() or Seek must have been called first.
	// This returns an error if the cursor is not found.
	Prev(clientID string, cursorKey string) (
		tv *msg.ThingValue, valid bool, err error)

	// PrevN moves the cursor to the previous N places from the current cursor
	// and return a list with N values in reverse time order.
	//
	//	clientID must match the owner of the cursor
	//	cursorKey is the cursor to iterate.
	//	until is the time to iterate towards
	//	limit is the maximum number of items to return. 0 defaults to DefaultLimit.
	//
	// This returns a list of values in reverse time order and 'itemsRemaining' which is
	// false if the iterator has reached the first match.
	PrevN(clientID string, cursorKey string, until time.Time, limit int) (
		tvList []*msg.ThingValue, itemsRemaining bool, err error)

	// ReadHistory returns the value history for the given thingID, affordance name and time range
	//
	//  thingID identifies the thing whose affordance data is stored
	//  affName is the affordance to read, or "" to read all stored props, events and actions
	//  timestamp is the start time to read from
	//  durationSec is the time period to read in seconds
	//  limit is the maximum number of values to read
	ReadHistory(thingID string, affName string, timestamp time.Time, durationSec int, limit int) (
		values []*msg.ThingValue, itemsRemaining bool, err error)

	// Release releases the cursor and frees its resources.
	// This invalidates all values obtained from the cursor
	ReleaseCursor(clientID string, cursorKey string) error

	// Seek positions the cursor at the given timestamp and cursor affordance name.
	// If the exact timestamp/affordance name is not found, the next item is returned.
	//
	// This returns the item found and a flag 'valid' if an item is found.
	// This returns an error if the cursor is not valid.
	Seek(clientID string, cursorKey string, ts time.Time) (
		tv *msg.ThingValue, valid bool, err error)
}

package server

import "github.com/hiveot/hivekit/go/modules/history"

// ReadHistoryServiceID is the thingID of the reading service
const ReadHistoryServiceID = "read"

// Management methods
const (
	// GetRetentionRuleMethod returns the first retention rule that applies
	// to the given value.
	GetRetentionRuleMethod = "getRetentionRule"

	// GetRetentionRulesMethod returns the collection of retention configurations
	GetRetentionRulesMethod = "getRetentionRules"

	// SetRetentionRulesMethod updates the set of retention rules
	SetRetentionRulesMethod = "setRetentionRules"
)

// Read history methods
const (
	// CursorNextNMethod returns a batch of next N historical values
	CursorNextNMethod = "cursorNextN"

	// CursorPrevNMethod returns a batch of prior N historical values
	CursorPrevNMethod = "cursorPrevN"

	// CursorReleaseMethod releases the cursor and resources
	// This MUST be called after the cursor is not longer used.
	CursorReleaseMethod = "cursorRelease"

	// CursorSeekMethod seeks the starting point in time for iterating the history
	// This returns a single value response with the value at timestamp or next closest
	// if it doesn't exist.
	// Returns empty value when there are no values at or past the given timestamp
	CursorSeekMethod = "cursorSeek"

	// GetCursorMethod returns a cursor to iterate the history of a things
	// The cursor MUST be released after use.
	// The cursor will expire after not being used for the default expiry period.
	GetCursorMethod = "getCursor"

	// ReadHistoryMethod reads the history up to a limit.
	ReadHistoryMethod = "readHistory"
)

// RetentionRuleSet is a map by event/action name with one or more rules for agent/things.
type RetentionRuleSet map[string][]*history.RetentionRule

type GetRetentionRuleArgs struct {
	// ThingID whose rule to get (digital twin ID) is optional
	ThingID string `json:"thingID,omitempty"`
	// Name of the event whose retention settings to get
	Name string `json:"name,omitempty"`
	// Retention for events,properties,actions or empty for all
	AffordanceType string `json:"affordanceType,omitempty"`
}
type GetRetentionRuleResp struct {
	Rule *history.RetentionRule `json:"rule"`
}

type GetRetentionRulesResp struct {
	Rules RetentionRuleSet `json:"rules"`
}

type SetRetentionRulesArgs struct {
	Rules RetentionRuleSet `json:"rules"`
}

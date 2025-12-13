package api

// Property, Event and Action affordance names as used in the interface
const (
	ActionCursor      = "cursor"
	ActionDelete      = "delete"
	ActionGet         = "get"
	ActionGetMultiple = "getMultiple"
	ActionSet         = "set"
	ActionSetMultiple = "setMultiple"
)

// SetArgs defines the arguments of the Set action
type SetArgs struct {

	// Document key to set
	Key string `json:"key"`

	// Doc is the serialized document to set
	Doc string `json:"value"`
}

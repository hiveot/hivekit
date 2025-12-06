package directory

import _ "embed"

// --- Constants ---

// DirectoryThingID is the default thingID of the directory module.
// Agents use this to publish events and subscribe to actions.
const DirectoryThingID = "directory"

// Default limit in retrieving things
const DefaultLimit = 300

// Property, Event and Action affordance names as used in the TM
const (
	PropThings              = "things"
	EventThingUpdated       = "thingUpdated"
	EventThingDeleted       = "thingDeleted"
	ActionCreateThing       = "createThing"
	ActionDeleteThing       = "deleteThing"
	ActionRetrieveThing     = "retrieveThing"
	ActionRetrieveAllThings = "retrieveAllThings"
	ActionUpdateThing       = "updateThing"
)

// RetrieveAllThingsArgs defines the arguments of the retrieveAllThings action
// Read all TDs - Read a batch of TD documents
type RetrieveAllThingsArgs struct {

	// Limit with Limit
	//
	// Maximum number of documents to return
	Limit int `json:"limit,omitempty"`

	// Offset with Offset
	//
	// Start index in the list of TD documents
	Offset int `json:"offset,omitempty"`
}

// RetrieveAllThingsResp response of the retrieveAllThings action
type RetrieveAllThingsOutput []string

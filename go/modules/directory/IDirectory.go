package directory

import "github.com/hiveot/hivekit/go/modules"

// DefaultDirectoryThingID is the default thingID of the directory module.
const DefaultDirectoryThingID = "directory"

// The http path that provides the TD of the service
// in case of the directory this provide the directory TD path
const WellKnownWoTPath = "/.well-known/wot"

// Default limit in retrieving things
const DefaultLimit = 300

// Property, Event and Action affordance names as used in the TM and messaging API
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

// RetrieveAllThingsOutput output of the retrieveAllThings action
type RetrieveAllThingsOutput []string

// IDirectoryModule defines the interface to the directory service module
// This is implemented in the service and the client api
type IDirectoryModule interface {
	modules.IHiveModule

	// CreateThing creates or updates the TD in the directory.
	// If the thing doesn't exist in the directory it is added.
	//
	// Things are stored under the ID of the agent.
	CreateThing(tdJson string) error

	// DeleteThing removes a Thing TD document from the directory
	DeleteThing(thingID string) error

	// RetrieveThing returns a JSON encoded TD document
	RetrieveThing(thingID string) (tdJSON string, err error)

	// RetrieveAllThings returns a batch of TD documents
	// This returns a list of JSON encoded digital twin TD documents
	RetrieveAllThings(offset int, limit int) (tdList []string, err error)

	// UpdateThing replaces the TD in the store.
	// If the thing doesn't exist in the store it is added.
	UpdateThing(tdJson string) error
}

package directory

// DefaultDirectoryThingID is the default thingID of the directory module.
const DefaultDirectoryThingID = "directory"

// Default limit in retrieving things
const DefaultLimit = 300

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

	// CreateThing creates or updates the TD in the directory.
	// If the thing doesn't exist in the directory it is added.
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

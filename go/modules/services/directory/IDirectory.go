package directory

// DefaultDirectoryThingID is the default thingID of the directory module.
const DefaultDirectoryThingID = "directory"

// Default limit in retrieving things
const DefaultLimit = 300

// IDirectory defines the interface to the directory service
type IDirectory interface {

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

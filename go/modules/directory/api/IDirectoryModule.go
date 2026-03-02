package directoryapi

import "github.com/hiveot/hivekit/go/modules"

// DefaultDirectoryServiceID is the default moduleID of the directory module instance.
// note that if multiple directory instances are created they must use different thingIDs
const DefaultDirectoryServiceID = "directory"

// The http path that provides the TD of the service
// in case of the directory this provide the directory TD path
const WellKnownWoTPath = "/.well-known/wot"

// Default limit in retrieving things
const DefaultLimit = 300

// IDirectoryServer defines the interface to the directory module server
// This is implemented in the service and the client api
type IDirectoryServer interface {
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

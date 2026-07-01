package directory

import (
	_ "embed"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/td"
)

// Embed the directory TM
//
//go:embed "directory-tm.json"
var DirectoryTMJson []byte

// two modules, the service and optional http server
const (
	// DirectoryServiceModuleType identifies the directory service module implementation
	DirectoryServiceModuleType = "DirectoryService"

	// DirectoryHttpModuleType identifies the http API module for the directory service
	// Place this module before any middleware so that requests are logged and authorized.
	DirectoryHttpModuleType = "directory-http"
)

// The thingID this directory identifies as for messaging. Must match the TD ID.
const DefaultDirectoryThingID = "thingDirectory"

// The http path that provides the TD of the service
// in case of the directory this provide the directory TD path
const WellKnownWoTPath = "/.well-known/wot"

// Default limit in retrieving things
const DefaultLimit = 300

// events, properties and actions
// these names must match the TD
const (
	ThingsProp        = "things"
	ThingUpdatedEvent = "thingUpdated"
	ThingDeletedEvent = "thingDeleted"
	CreateThingAction = "createThing"
	DeleteThingAction = "deleteThing"
	// another module in the chain can retrieve the directory TDD using this action
	RetrieveTDDAction       = "retrieveTDD"
	RetrieveThingAction     = "retrieveThing"
	RetrieveAllThingsAction = "retrieveAllThings"
	UpdateThingAction       = "updateThing"
)

// The handler of TD write requests
// This returns the original or a modified TD
// This returns an error if writing the TD is not allowed.
//
// clientID is the account ID of the client writing the TD, eg the device.
type WriteTDHook func(clientID string, tdoc *td.TD) (*td.TD, error)

// The handler of deleting TD requests
// This returns an error if deleting the TD is not allowed.
//
// clientID is the account ID of the client writing the TD, eg the device.
type DeleteTDHook func(clientID string, thingID string) error

// Information of the thing registrations for a client account.
type RegistrationInfo struct {
	// The clientID of the device managing the Things.
	ClientID string
	// ThingIDs of the devices it has registered
	ThingIDs []string
}

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

// Directory http server as per https://w3c.github.io/wot-discovery/#exploration-directory-api
// This acts as a simple http transport server and should be placed ahead of
// the DirectoryService module chain.
type IDirectoryHttpServer interface {
	api.ITransportServer
}

// IDirectoryService defines the interface to the directory service module
type IDirectoryService interface {
	api.IHiveModule

	// CreateThing creates or updates the TD in the directory.
	// If the thing doesn't exist in the directory it is added.
	//
	// Only devices can create the TD of things that use reverse connections.
	//
	// Administrators can upload TDs using their own account but only if these
	// devices do not use reverse connections.
	CreateThing(senderID string, tdJson string) error

	// DeleteThing removes a Thing TD document from the directory
	DeleteThing(senderID string, thingID string) error

	// Return an instance of a TD from the store.
	// These TD's are cached so successive requests do not parse the json each time.
	GetTD(thingID string) *td.TD

	// Return the Directory TD and its json
	GetTDD() (*td.TD, string)

	// RetrieveThing returns a JSON encoded TD document
	RetrieveThing(thingID string) (tdJSON string, err error)

	// RetrieveAllThings returns a batch of TD documents
	// This returns a list of JSON encoded digital twin TD documents
	RetrieveAllThings(offset int, limit int) (tdList []string, err error)

	// Install a hook that is called when a Thing is writing its TD to the directory.
	// This hook returns the TD that is actually written.
	// Intended for updating forms and for supporting the digital twin concept.
	//
	//  thingID is the ID whose thing is written.
	//  tdi is the TD instance, or nil if the thing TD is deleted.
	SetTDHooks(writeTDHandler WriteTDHook, deleteTDHandler DeleteTDHook)

	// UpdateThing replaces the TD in the store.
	// If the thing doesn't exist in the store it is added.
	UpdateThing(senderID string, tdJson string) error
}

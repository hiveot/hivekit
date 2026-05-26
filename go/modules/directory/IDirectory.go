package directory

import (
	_ "embed"

	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/transport"
)

// Embed the directory TM
//
//go:embed "directory-tm.json"
var DirectoryTMJson []byte

// DirectoryModuleType identifies the directory module implementation
const DirectoryModuleType = "directory"

// DirectoryHttpModuleType identifies the http API module for the directory service
// Place this module before any middleware so that requests are logged and authorized.
const DirectoryHttpModuleType = "directory-http"

// The thingID this directory identifies as for messaging. Must match the TD ID.
const DefaultDirectoryThingID = "thingDirectory"

// The http path that provides the TD of the service
// in case of the directory this provide the directory TD path
const WellKnownWoTPath = "/.well-known/wot"

// Default limit in retrieving things
const DefaultLimit = 300

// The handler of TD write requests
// This returns the original or a modified TD
// This returns an error if writing the TD is not allowed.
type WriteTDHook func(agentID string, tdi *td.TD) (*td.TD, error)

// The handler of deleting TD requests
// This returns an error if deleting the TD is not allowed.
type DeleteTDHook func(agentID string, thingID string) error

// Information of an agent that has registered Things
type AgentInfo struct {
	// The agent whose info this contains
	AgentID string
	// ThingIDs of the devices it has registered
	ThingIDs []string
}

// Directory http server as per https://w3c.github.io/wot-discovery/#exploration-directory-api
// This acts as a simple http transport server and should be placed ahead of
// the DirectoryService module chain.
type IDirectoryHttpServer interface {
	transport.ITransportServer
}

// IDirectoryService defines the interface to the directory service module
type IDirectoryService interface {
	modules.IHiveModule

	// CreateThing creates or updates the TD in the directory.
	// If the thing doesn't exist in the directory it is added.
	//
	// Only agents can create a TD. If the administrator acts as the agent then it
	// is also responsible for updating it if that is ever needed.
	CreateThing(agentID string, tdJson string) error

	// DeleteThing removes a Thing TD document from the directory
	DeleteThing(agentID string, thingID string) error

	// Return an instance of a TD from the store.
	// These TD's are cached so successive requests do not parse the json each time.
	GetTD(thingID string) *td.TD

	// Retrieve the directory TD
	RetrieveTDD() string

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
	//
	// Only agents can update a TD.
	UpdateThing(agentID string, tdJson string) error
}

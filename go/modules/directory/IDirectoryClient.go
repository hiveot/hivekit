package directory

import (
	_ "embed"

	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules"
)

const DirectoryClientModuleType = "DirectoryClient"

// IDirectoryCache defines the interface of the local cache of TDs
type IDirectoryCache interface {

	// Get all available Thing TDs from the local cache.
	//
	// Offset is the offset in the list for iteration.
	// Limit is the maximum number of things to return.
	GetAllThings(offset int, limit int) []*td.TD

	// Get a Thing TD from the local cache.
	//
	// This returns nil if the TD is not in the local cache.
	GetThing(thingID string) *td.TD

	// Import the TD into the directory client cache.
	// Intended for discovered things or out of band cache loading.
	ImportTD(tdJSON string) (*td.TD, error)
}

// IDirectoryClient defines the interface to the directory consumer.
//
// The directory client retrieves TDs from a discovered or configured directory server
// and can load TDs from a individual discovered Things. See also SetDirectory(tdd)
//
// Note that CreateThing and UpdateThing are not supported as these are methods
// intended for devices and services, not consumers.

type IDirectoryClient interface {
	modules.IHiveModule

	// CreateThing creates or updates the TD in the directory.
	// If the thing doesn't exist in the directory it is added.
	//
	// Only devuces and services can create a TD. If the administrator acts as the device then it
	// is also responsible for updating it if that is ever needed.
	// CreateThing(tdJson string) error

	// Return the local cache of things
	Cache() IDirectoryCache

	// DeleteThing removes a Thing TD document from the cache and the remote directory.
	// Sufficient access rights are required for the remote directory.
	//
	// This returns an error on insufficient permissions.
	DeleteThing(thingID string) error

	// RetrieveAllThings loads a batch of TD JSON documents from the directory server
	// and updates the local cache.
	//
	// This returns a list of TD JSON documents
	RetrieveAllThings(offset int, limit int) (tdList []*td.TD, err error)

	// RetrieveThing loads a TD document from the directory server and updates the local cache.
	//
	// If the TD already exists in the local cache then it is returned instead.
	//
	// This updates the TD in the local cache and returns the server provided JSON document.
	RetrieveThing(thingID string) (tdoc *td.TD, err error)

	// Manually set the TDD of the directory server.
	// Needed to send requests to the directory server.
	SetTDD(tdoc *td.TD)
}

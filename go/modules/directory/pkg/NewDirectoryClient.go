package directorypkg

import (
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/directory"
	directoryclient "github.com/hiveot/hivekit/go/modules/directory/internal/client"
	"github.com/hiveot/hivekit/go/modules/factory"
)

// NewDirectoryClient creates a client for the Thing directory service.
//
// This client can also be used stand-alone without a directory server. In this case
// it can be configured to read TD's from the local file system. By copying TD JSON
// files into this folder the directory client can be 'pre-charged' out-of-band with TDs.
//
// The local filesystem can also be used to bootstrap the directory TDD by providing
// the directory TDD file named "directory.json".
//
// An in-memory cache is used to speed up queries. If a TD is in the in-memory cache the
// requested TD is returned. If no TD is cached and the directory server is available
// then a query is send to the server. If no server is available the local filesystem
// is read for a file named {ThingID}.json.
//
// If a search for a TD fails then a nil TD is added to the local cache to speed up
// future queries.
//
// If a directory server connection is available using a bi-directional protocol then
// the client subscribes to updates from the directory server to ensure the TD's remain
// up to date. If a missing TD is added the notification will ensure that the local
// cache is updated.
//
// If a directory server is unknown or not reachable the local cache can use a file
// storage for out-of-band configuration of TDs. TDs on the local filesystems are
// read-only and only intended for oob configuration..
//
// This implements the IDirectory and IDirectoryCache interface.
//
// # Use with a Factory Recipe:
//
// When used in a factory recipe together with discovery the recommended sequence is:
//
//	consumer - chain[ discoveryClient - directoryClient - router | client ]
//
// If no directory TDD is configured the discovery client automatically initiates
// a search for the directory server on Start and updates the app environment with
// the directory server TDD.
//
// The directoryClient factory retrieves the server TDD from the app environment.
// If no TDD is available on start, the directory checks file storage.
//
// Once the recipe has been started. the consumer can locate the directory client
// in the factory and use it to determine available Things.
//
// The directory client is essential for bootstrapping the client as it provides the TD's
// of devices to interact with. This bootstrap process requires a TDD of the directory
// server. This can be set manually or using discovery.
//
// Since its so essential for WoT interaction, the factory has a field that holds the
// TDD that is used when this client is instantiated through the factory.
//
// When not using the factory, the TDD can be obtained using discovery.
//
//	dirTDD is the directory TD from external source on start.
//	See also the discovery client which supports this method.
//
// This returns a new instance of the directory client
func NewDirectoryClient(dirTDD *td.TD, sink modules.IHiveModule) directory.IDirectoryClient {
	dirClient := directoryclient.NewDirectoryClientImpl(dirTDD, sink)
	return dirClient
}

// NewDirectoryClientFactory creates the directory client using the TDD from the app environment.
// If no TDD is available then Start checks for an out-of-band stored TDD file.
func NewDirectoryClientFactory(f factory.IModuleFactory, modDef *factory.ModuleDefinition) (modules.IHiveModule, error) {
	appEnv := f.GetEnvironment()
	dirClient := NewDirectoryClient(appEnv.DirTDD, nil)
	return dirClient, nil
}

// // Create the new directory client for the http protocol as per spec
// func NewDirectoryHttpClient(dirTD *td.TD, caCert *x509.Certificate) directory.IDirectoryClient {
// 	dirClient := directoryclient.NewDirectoryHttpClient(dirTD, caCert)
// 	return dirClient
// }

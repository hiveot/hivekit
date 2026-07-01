package directoryclient

import (
	"fmt"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/directory"
	"github.com/hiveot/hivekit/go/modules/transport/discovery"
)

// Implementation of the directory client
//
// TODO:
// 1. import tds from discovery
// 2. load tdd from file
type DirectoryClientImpl struct {
	*modules.HiveModuleBase

	// TBD: should the directory cache support the filesystem for out-of-band TDs?
	cache *DirectoryCacheImpl

	// discoveryThingID ThingID of the directory service instance.
	discoveryThingID string

	// the retrieved directory TDD used to connect to the directory server
	dirTDD *td.TD
}

// Return the local cache of Things
func (cl *DirectoryClientImpl) Cache() directory.IDirectoryCache {
	return cl.cache
}

// Send request to delete a TD
func (cl *DirectoryClientImpl) DeleteThing(thingID string) error {
	cl.cache.RemoveTD(thingID)

	// the senderID is added by the transport server
	err := cl.Rpc(td.OpInvokeAction,
		cl.dirTDD.ID, directory.DeleteThingAction, thingID, nil)
	return err
}

// Receive notifications from the directory service to update the directory
func (cl *DirectoryClientImpl) HandleNotification(notif *msg.NotificationMessage) {
	cl.HiveModuleBase.HandleNotification(notif)
}

// Read the TDD from a configuration file (not yet supported)
func (cl *DirectoryClientImpl) readTDDFromFile() (*td.TD, error) {
	return nil, fmt.Errorf("Read TDD from file is not yet supported")
}

func (cl *DirectoryClientImpl) RetrieveThing(thingID string) (tdoc *td.TD, err error) {

	// first try the cache
	tdoc = cl.cache.GetThing(thingID)
	if tdoc != nil {
		return tdoc, nil
	}

	//retrieve the TD and update the local cache
	var tdJson string
	err = cl.Rpc(td.OpInvokeAction,
		cl.dirTDD.ID, directory.RetrieveThingAction, thingID, &tdJson)
	if err != nil {
		return nil, err
	}
	tdoc, err = cl.cache.ImportTD(tdJson)
	return tdoc, err
}

func (cl *DirectoryClientImpl) RetrieveAllThings(offset int, limit int) (tdList []*td.TD, err error) {

	args := directory.RetrieveAllThingsArgs{
		Offset: offset,
		Limit:  limit,
	}
	var tdJsonList []string
	err = cl.Rpc(td.OpInvokeAction,
		cl.dirTDD.ID, directory.RetrieveAllThingsAction, args, &tdJsonList)

	// import them into the cache
	tdList = make([]*td.TD, 0, len(tdJsonList))
	for _, tdJson := range tdJsonList {
		tdoc, err := cl.cache.ImportTD(tdJson)
		if err == nil {
			tdList = append(tdList, tdoc)
		}
	}
	return tdList, err
}

// Set the directory TD to use.
func (cl *DirectoryClientImpl) SetTDD(tdd *td.TD) {
	cl.dirTDD = tdd
}

// Start the directory client and retrieve the TDD.
//
// For the directory client to function it needs a directory server TDD.
// If non is provided on instantiation then check the filesystem for an out-of-band
// configured TDD.
//
// Start fails if no TDD is found.
func (cl *DirectoryClientImpl) Start() (err error) {

	if cl.dirTDD == nil {
		cl.dirTDD, err = cl.readTDDFromFile()
	}
	return err
}

// Update a Thing TD in the directory and wait for confirmation
// This retuns nil if success or an error if something went wrong.
// func (cl *DirectoryMsgClient) UpdateTD(tdJson string) error {

// 	req := msg.NewRequestMessage(
// 		td.OpInvokeAction, cl.directoryID, directory.ActionUpdateThing, tdJson, "")
// 	_, err := cl.ForwardRequestWait(req)

// 	return err
// }

// NewDirectoryClientImpl creates a new DirectoryClient instance for consumers which
// uses RRN messages for communicating with the directory server.
//
// Use the sink to link to a transport client for delivering the request. Note that
// the transport client must be provided the directory instance to be able to get the
// TDs of the destination.
//
// Tip: This client can be used as the directory for the Router Module. Set the router
// module as the sink (or somewhere else downstream) and provide this instance when
// creating the router. Last, add the directory TDD with LoadTD(tdd) so that the
// router knows how to connect to the directory server when receiving a request.
//
// Devices should use the directorypkg.UpdateTD function to publish their TD(s) to
// the discovery or directory server.
//
// This listens for directory notifications from the sink to receive directory updates.
//
//	dirTDD is the optional directory TD from external source.
//	sink forwards requests to the directory server and returns notifications. nil to set manually.
func NewDirectoryClientImpl(dirTDD *td.TD, sink api.IHiveModule) *DirectoryClientImpl {

	cl := &DirectoryClientImpl{
		HiveModuleBase:   modules.NewHiveModuleBase("", 0),
		cache:            NewDirectoryCacheImpl(),
		dirTDD:           dirTDD,
		discoveryThingID: discovery.DiscoveryClientModuleType,
	}
	if sink != nil {
		cl.SetRequestSink(sink)
		// notifications returned are passed to this client (if any subscriptions are made)
		sink.SetNotificationSink(cl)
	}
	var _ directory.IDirectoryClient = cl // interface check
	return cl
}

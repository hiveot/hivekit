package clientimpl

import (
	"fmt"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/cells"
	"github.com/hiveot/hivekit/go/cells/directory"
	"github.com/hiveot/hivekit/go/cells/transport/discovery"
	"github.com/teris-io/shortid"
)

// Implementation of the directory client
//
// TODO:
// 1. import tds from discovery
// 2. load tdd from file
type DirectoryClientImpl struct {
	*cells.HiveCellBase

	// configuration folder potentially containing a TDD file
	configDir string

	// TBD: should the directory cache support the filesystem for out-of-band TDs?
	cache *DirectoryCacheImpl

	// discoveryThingID ThingID of the directory service instance.
	discoveryThingID string

	// the retrieved directory TDD used to connect to the directory server
	dirTDD *td.TD
}

// Send a request to the directory server.
//
// Use the TDD ThingID if known. Otherwise fall back to the default directory ThingID.
func (cl *DirectoryClientImpl) _sendServerRequest(
	op string, action string, input any, output any) error {

	var dirID = directory.DefaultDirectoryThingID

	if cl.dirTDD != nil {
		dirID = cl.dirTDD.ID
	}
	// this assumes that the chain knows how to reach the directory server.
	// This is not a concern of this cell though.
	err := cl.Rpc(op, dirID, action, input, output)
	if err != nil {
		return fmt.Errorf("RetrieveAllThings: op '%s' failed: %w", op, err)
	}
	return err
}

// Return the local cache of Things
func (cl *DirectoryClientImpl) Cache() directory.IDirectoryCache {
	return cl.cache
}

// Send request to delete a TD
// If no TDD is set then this removes the TD from the cache and an error is returned.
func (cl *DirectoryClientImpl) DeleteThing(thingID string) (err error) {
	cl.cache.RemoveTD(thingID)

	// This client doesnt make assumptions on how it is connected.
	// If the cell downstream is connected to a gateway then this will work, otherwise it a TDD is required.
	err = cl._sendServerRequest(td.OpInvokeAction, directory.DeleteThingAction, thingID, nil)
	return err
}

// Get the directory TD to client is using to talk to the remote directory
func (cl *DirectoryClientImpl) GetTDD() *td.TD {
	return cl.dirTDD
}

// Receive notifications from the directory service to update the directory
// TODO:
// 1. update TD from directory events
// 2. handle TDD discovery notification and subscribe to the directory server
func (cl *DirectoryClientImpl) HandleNotification(notif *msg.NotificationMessage) {
	cl.HiveCellBase.HandleNotification(notif)
}

// Retrieve a Thing TD from the cache or remote
func (cl *DirectoryClientImpl) RetrieveThing(thingID string) (tdoc *td.TD, err error) {

	// first try the cache
	tdoc = cl.cache.GetThing(thingID)
	if tdoc != nil {
		return tdoc, nil
	}

	// This client doesnt make assumptions on how it is connected.
	// If the cell downstream is connected to a gateway then this will work, otherwise it a TDD is required.
	var tdJson string
	err = cl._sendServerRequest(
		td.OpInvokeAction, directory.RetrieveThingAction, thingID, &tdJson)
	if err != nil {
		return nil, err
	}
	tdoc, err = cl.cache.ImportTDJson(tdJson)
	return tdoc, err
}

// Retrieve all things in the directory
// This fails if the TDD is not set.
func (cl *DirectoryClientImpl) RetrieveAllThings(offset int, limit int) (tdList []*td.TD, err error) {

	// This client doesnt make assumptions on how it is connected.
	// If the cell downstream is connected to a gateway then this will work, otherwise it a TDD is required.

	args := directory.RetrieveAllThingsArgs{
		Offset: offset,
		Limit:  limit,
	}
	var tdJsonList []string
	err = cl._sendServerRequest(
		td.OpInvokeAction, directory.RetrieveAllThingsAction, args, &tdJsonList) //&tdJsonList)
	if err != nil {
		return nil, err
	}

	// import them into the cache
	tdList = make([]*td.TD, 0, len(tdJsonList))
	for _, tdJson := range tdJsonList {
		tdoc, err := cl.cache.ImportTDJson(tdJson)
		if err == nil {
			tdList = append(tdList, tdoc)
		}
	}
	return tdList, err
}

// Set the directory TD to use and include it in the local cache
func (cl *DirectoryClientImpl) SetTDD(tdd *td.TD) {
	cl.dirTDD = tdd
	cl.cache.ImportTD(tdd)
}

// NewDirectoryClientImpl creates a ready-to-use DirectoryClient instance for consumers which
// uses RRN messages for communicating with the directory server.
//
// Use the sink to link to a transport client for delivering the request. Note that
// the transport client must be provided the directory instance to be able to get the
// TDs of the destination.
//
// Tip: This client can be used as a directory cache for the Router. Set the router
// as the sink (or somewhere else downstream) and provide this instance when
// creating the router. Last, add the directory TDD with LoadTD(tdd) so that the
// router knows how to connect to the directory server when receiving a request.
//
// Devices should use the UpdateTD function to publish their TD(s) to the discovery
// or directory server.
//
// This listens for directory notifications from the sink to receive directory updates.
//
//	dirTDD is the optional directory TD from external source. Use SetTDD if not yet available.
//	reqSink forwards requests to the directory server and returns notifications. nil to set manually.
func NewDirectoryClientImpl(dirTDD *td.TD, reqSink api.IHiveCell) *DirectoryClientImpl {
	thingID := directory.DirectoryClientCellType + "-" + shortid.MustGenerate()
	cl := &DirectoryClientImpl{
		HiveCellBase:     cells.NewHiveCellBase(thingID, 0),
		cache:            NewDirectoryCacheImpl(),
		dirTDD:           dirTDD,
		discoveryThingID: discovery.DiscoveryClientCellType,
	}
	if reqSink != nil {
		cl.SetRequestSink(reqSink)
		// notifications returned are passed to this client (if any subscriptions are made)
		reqSink.SetNotificationSink(cl)
	}

	// TODO: support for a TDD cache?
	// if dirTDD == nil {
	// 	dirTDDPath := path.Join(configDir, directory.ConfigTDDFilename)
	// 	dirTDD, err = td.ReadTDFromFile(dirTDDPath)
	// 	// not having a TDD is not fatal
	// 	err = nil
	// }
	var _ directory.IDirectoryClient = cl // interface check
	return cl
}

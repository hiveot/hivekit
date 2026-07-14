package directoryclient

import (
	"fmt"
	"path"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/directory"
	"github.com/hiveot/hivekit/go/modules/transport/discovery"
	"github.com/teris-io/shortid"
)

// Implementation of the directory client
//
// TODO:
// 1. import tds from discovery
// 2. load tdd from file
type DirectoryClientImpl struct {
	*modules.HiveModuleBase

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
func (m *DirectoryClientImpl) _sendServerRequest(
	op string, action string, input any, output any) error {

	var dirID = directory.DefaultDirectoryThingID

	if m.dirTDD != nil {
		dirID = m.dirTDD.ID
	}
	// this assumes that the client knows how to reach the directory. This is not a concern
	// of this module though.
	err := m.Rpc(op, dirID, action, input, output)
	if err != nil {
		return fmt.Errorf("RetrieveAllThings: op '%s' no directory connection", op)
	}
	return err
}

// Return the local cache of Things
func (m *DirectoryClientImpl) Cache() directory.IDirectoryCache {
	return m.cache
}

// Send request to delete a TD
// If no TDD is set then this removes the TD from the cache and an error is returned.
func (m *DirectoryClientImpl) DeleteThing(thingID string) (err error) {
	m.cache.RemoveTD(thingID)

	// This client doesnt make assumptions on how it is connected.
	// If the module downstream is connected to a gateway then this will work, otherwise it a TDD is required.
	err = m._sendServerRequest(td.OpInvokeAction, directory.DeleteThingAction, thingID, nil)
	return err
}

// Receive notifications from the directory service to update the directory
func (m *DirectoryClientImpl) HandleNotification(notif *msg.NotificationMessage) {
	m.HiveModuleBase.HandleNotification(notif)
}

// Retrieve a Thing TD from the cache or remote
func (m *DirectoryClientImpl) RetrieveThing(thingID string) (tdoc *td.TD, err error) {

	// first try the cache
	tdoc = m.cache.GetThing(thingID)
	if tdoc != nil {
		return tdoc, nil
	}

	// This client doesnt make assumptions on how it is connected.
	// If the module downstream is connected to a gateway then this will work, otherwise it a TDD is required.
	var tdJson string
	err = m._sendServerRequest(
		td.OpInvokeAction, directory.RetrieveThingAction, thingID, &tdJson)
	if err != nil {
		return nil, err
	}
	tdoc, err = m.cache.ImportTDJson(tdJson)
	return tdoc, err
}

// Retrieve all things in the directory
// This fails if the TDD is not set.
func (m *DirectoryClientImpl) RetrieveAllThings(offset int, limit int) (tdList []*td.TD, err error) {

	// This client doesnt make assumptions on how it is connected.
	// If the module downstream is connected to a gateway then this will work, otherwise it a TDD is required.

	args := directory.RetrieveAllThingsArgs{
		Offset: offset,
		Limit:  limit,
	}
	var tdJsonList []string
	err = m._sendServerRequest(
		td.OpInvokeAction, directory.RetrieveAllThingsAction, args, &tdJsonList)
	if err != nil {
		return nil, err
	}

	// import them into the cache
	tdList = make([]*td.TD, 0, len(tdJsonList))
	for _, tdJson := range tdJsonList {
		tdoc, err := m.cache.ImportTDJson(tdJson)
		if err == nil {
			tdList = append(tdList, tdoc)
		}
	}
	return tdList, err
}

// Set the directory TD to use.
func (m *DirectoryClientImpl) SetTDD(tdd *td.TD) {
	m.dirTDD = tdd
}

// Start the directory client and retrieve the TDD.
//
// For the directory client to function it needs a directory server TDD.
// If non is provided on instantiation then check the filesystem for an out-of-band
// configured TDD.
//
// Start fails if no TDD is found.
func (m *DirectoryClientImpl) Start() (err error) {

	if m.dirTDD == nil {
		dirTDDPath := path.Join(m.configDir, directory.ConfigTDDFilename)
		m.dirTDD, err = td.ReadTDFromFile(dirTDDPath)
		// not having a TDD is not fatal
		err = nil
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
//	dirTDD is the optional directory TD from external source. Use SetTDD if not yet available.
//	sink forwards requests to the directory server and returns notifications. nil to set manually.
func NewDirectoryClientImpl(dirTDD *td.TD, sink api.IHiveModule) *DirectoryClientImpl {

	thingID := directory.DirectoryClientModuleType + "-" + shortid.MustGenerate()
	cl := &DirectoryClientImpl{
		HiveModuleBase:   modules.NewHiveModuleBase(thingID, 0),
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

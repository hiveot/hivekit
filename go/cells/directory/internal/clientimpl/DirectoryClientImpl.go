package clientimpl

import (
	"fmt"
	"log/slog"

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
// Tip: This client can be used as a directory cache for the Router so the router
// can determine how to connect to remote things. Manually: Provide the router
// with the GetTD method provided by this directory. Factory: use a recipe that
// includes the directory service or client. The router factory function will
// look for a either cell (DirectoryServiceCellType or DirectoryClientCellType).
//
// This client is intended for consumers. Devices should use the standalone UpdateTD
// method to publish their TD(s) to the discovery or directory server.
type DirectoryClientImpl struct {
	*cells.HiveCellBase

	// configuration folder potentially containing a TDD file
	configDir string

	// TBD: should the directory cache support the filesystem for out-of-band TDs?
	cache *DirectoryCacheImpl

	// discoveryThingID ThingID of the directory service instance.
	discoveryThingID string

	// the directory TD used to connect to the directory server
	dirTD *td.TD
}

// Send a request to the directory server.
//
// If no TDD is set then just send the request without it. In case of a gateway
// connection, it goes straight to the directory service.
//
// Use the TDD ThingID if known. Otherwise fall back to the default directory ThingID.
func (cl *DirectoryClientImpl) _sendServerRequest(
	op string, action string, input any, output any) error {

	var dirThingID string

	if cl.dirTD != nil {
		dirThingID = cl.dirTD.ID
	} else {
		// this is only going to work if the chain leads to the directory service
		slog.Info("_sendServerRequest. Directory client has no TDD. Forwarding request anyways",
			"op", op, "action", action)
	}

	// this assumes that the chain knows how to reach the directory server.
	// This is not a concern of this cell though.
	err := cl.Rpc(op, dirThingID, action, input, output)
	if err != nil {
		return fmt.Errorf("RetrieveAllThings: op '%s' failed: %w", op, err)
	}
	return err
}

// Return the local cache of Things
func (cl *DirectoryClientImpl) Cache() directory.IDirectoryCache {
	return cl.cache
}

// Create a Thing in the directory
func (cl *DirectoryClientImpl) CreateThing(tdJson string) (err error) {
	_, err = cl.cache.ImportTDJson(tdJson)
	if err != nil {
		return err
	}

	// This client doesnt make assumptions on how it is connected.
	// If a sink downstream is connected to a gateway then this will work, otherwise it a TDD is required.
	err = cl._sendServerRequest(
		td.OpInvokeAction, directory.CreateThingAction, tdJson, nil)
	return err
}

// Send request to delete a TD
// If no TDD is set then this removes the TD from the cache and an error is returned.
func (cl *DirectoryClientImpl) DeleteThing(thingID string) (err error) {
	cl.cache.RemoveTD(thingID)

	// This client doesnt make assumptions on how it is connected.
	// If the cell downstream is connected to a gateway then this will work, otherwise it a TDD is required.
	err = cl._sendServerRequest(
		td.OpInvokeAction, directory.DeleteThingAction, thingID, nil)
	return err
}

// Get the TD for the given thing ID
func (cl *DirectoryClientImpl) GetTD(thingID string) *td.TD {
	tdoc, err := cl.RetrieveThing(thingID)
	_ = err
	return tdoc
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
	cl.dirTD = tdd
	cl.cache.ImportTD(tdd)
}

// Update a Thing in the directory
func (cl *DirectoryClientImpl) UpdateThing(tdJson string) (err error) {
	_, err = cl.cache.ImportTDJson(tdJson)
	if err != nil {
		return err
	}
	err = cl._sendServerRequest(td.OpInvokeAction, directory.UpdateThingAction, tdJson, nil)
	return err
}

// NewDirectoryClientImpl creates a ready-to-use DirectoryClient instance for consumers which
// uses RRN messages for communicating with the directory server.
//
// Use the sink to link to a transport client for delivering directory requests.
//
// Note that the transport client needs a TD to establish a connection, or an
// existing connection to forward directory requests.
//
// This listens for directory notifications from the sink to receive directory updates.
//
//	dirTD is the required directory TD from external source. Use SetTDD if not yet available.
//	reqSink forwards requests to the directory server and returns notifications. nil to set manually.
func NewDirectoryClientImpl(
	dirTD *td.TD, reqSink api.IHiveCell) *DirectoryClientImpl {
	thingID := directory.DirectoryClientCellType + "-" + shortid.MustGenerate()
	cl := &DirectoryClientImpl{
		HiveCellBase:     cells.NewHiveCellBase(thingID),
		cache:            NewDirectoryCacheImpl(),
		dirTD:            dirTD,
		discoveryThingID: discovery.DiscoveryClientCellType,
	}
	if dirTD != nil {
		cl.SetTDD(dirTD)
	}
	if reqSink != nil {
		cl.SetRequestSink(reqSink)
		// notifications returned are passed to this client (if any subscriptions are made)
		reqSink.SetNotificationSink(cl)
	}

	// TODO: support for loading TDD from file?
	// if dirTDD == nil {
	// 	dirTDDPath := path.Join(configDir, directory.ConfigTDDFilename)
	// 	dirTDD, err = td.ReadTDFromFile(dirTDDPath)
	// 	// not having a TDD is not fatal
	// 	err = nil
	// }
	var _ directory.IDirectoryClient = cl // interface check
	return cl
}

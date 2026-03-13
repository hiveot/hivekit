package module

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/bucketstore"
	bucketstoreapi "github.com/hiveot/hivekit/go/modules/bucketstore/api"
	digitwinapi "github.com/hiveot/hivekit/go/modules/digitwin/api"
	"github.com/hiveot/hivekit/go/modules/directory"
	directoryapi "github.com/hiveot/hivekit/go/modules/directory/api"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/modules/vcache"
	vcacheapi "github.com/hiveot/hivekit/go/modules/vcache/api"
	"github.com/hiveot/hivekit/go/msg"
	"github.com/hiveot/hivekit/go/wot"
	"github.com/hiveot/hivekit/go/wot/td"
)

// DefaultDigitwinServiceID is the default moduleID of the digital twin module instance.
const DefaultDigitwinServiceID = "digitwin"

// DigitwinModule implements the digital twin module.
//
// This module serves a Digital Twin for eligible devices. It hooks into the provided
// thing directory to replace the device TD with a digital twin.
// It subscribes to notifications from registered devices and handles read and write requests for the
// digital twin. Where neccesary it forwards requests to the actual device if reachable.
//
// This tracks the TDs of the original devices in a separate device directory for use
// by modules like the router.
type DigitwinModule struct {
	modules.HiveModuleBase

	// hook to server to add forms to a TD for interacting with affordances
	addForms func(tdoc *td.TD, includeAffordances bool)

	// internal storage with the original TDs
	bucket      bucketstoreapi.IBucket
	bucketName  string
	bucketStore bucketstoreapi.IBucketStore

	// the device directory holding TD's of the native devices/agents
	deviceDirectory directoryapi.IDirectoryServer

	// the Thing directory with digital twin TDs
	// this also contains TDs of non-digital twin devices and services, as consumers
	// should be able to use these as well.
	directory directoryapi.IDirectoryServer

	// the store that holds the digital twin TDs and value
	digitwinStore bucketstoreapi.IBucketStore

	// configuration to add forms to all the affordances of a TD
	includeAffordanceForms bool

	// the RRN messaging API for the digitwin module itself
	msgAPI *DigitwinMsgHandler

	// notification cache holding device property and events values
	vcache vcacheapi.IVCacheModule

	// location of the digital twin storage area
	storageRoot string
}

// ForwardDigitalTwinRequest passes the request made to a digital twin to the original device.
// This will restore the original device thingID before forwarding the request.
func (m *DigitwinModule) ForwardDigitwinRequestToDevice(dtwReq *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
	// reverse the digital twin thingID
	agentID, thingID, err := SplitDigitwinID(dtwReq.ThingID)

	// the device agent expects the actual thingID
	deviceReq := *dtwReq
	deviceReq.ThingID = thingID

	// forward the request to the sink, which is responsible for routing it to the destination
	_ = agentID
	err = m.ForwardRequest(&deviceReq, func(resp *msg.ResponseMessage) error {
		// put the digitwin thingID back into the response
		resp.ThingID = dtwReq.ThingID
		err = replyTo(resp)
		return err
	})
	return err
}
func (m *DigitwinModule) GetDeviceDirectory() directoryapi.IDirectoryServer {
	return m.deviceDirectory
}

// HandleNotification stores the latest notification things for retrieval as a digital twin value
func (m *DigitwinModule) HandleNotification(notif *msg.NotificationMessage) {

	// track online status of agents - this needs tracking of agents
	// agentInfo := m.deviceDirectory.GetAgent(notif.SenderID)
	if notif.Name == transports.ConnectedEventName {
		// if this is an agent then subscribe to notifications
		// actually this isn't needed as agents publish all notifications anyways
		// if agentInfo != nil {
		// slog.Info("Agent connected", slog.String("agentID", notif.SenderID))
		// }
	} else if notif.Name == transports.DisconnectedEventName {
		// if this is an agent then its things are no longer online
		// if agentInfo != nil {
		// slog.Info("Agent disconnected", slog.String("agentID", notif.SenderID))
		// }
	}
	// 1: is this a digital twin not
	dtwNotif := *notif
	dtwNotif.ThingID = MakeDigitwinID(notif.SenderID, notif.ThingID)
	m.vcache.HandleNotification(&dtwNotif)

	m.ForwardNotification(notif)
}

// HandleRequest for digital twins requests.
// This looks at the dtw prefix to determine if this is a digital twin.
//
// - handle read requests directly from cache
// - route write requests to device
// - route action requests to device
//
// This invokes the replyTo response handler with a response.
//
// If the request is not for this module then it is forwarded to the next sink.
// If the request is for this module but invalid, an error is returned
func (m *DigitwinModule) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {

	// Handle requests for a digital twin
	if strings.HasPrefix(req.ThingID, digitwinapi.DigitwinIDPrefix) {
		switch req.Operation {

		// read requests are handled by the value cache
		// the value cache holds digital twin property and event notifications

		// FIXME: unobservable properties will never be in the cache. If the value
		// isn't cached the request should be forwarded. Eg set vcache sink to the
		// server or client connection that can forward it.
		case wot.OpReadAllProperties,
			wot.OpReadMultipleProperties,
			wot.OpReadProperty,
			wot.HTOpReadEvent,
			wot.HTOpReadAllEvents:
			return m.vcache.HandleRequest(req, replyTo)

		// write requests are forwarded to the actual device after mapping
		// the thingID back to that of the device
		case wot.OpWriteProperty,
			wot.OpWriteMultipleProperties,
			wot.OpInvokeAction:
			return m.ForwardDigitwinRequestToDevice(req, replyTo)
		}
	}

	// Handle requests for this module
	if req.ThingID != m.GetModuleID() {
		return nil
	} else if req.SenderID == "" {
		err := fmt.Errorf("missing senderID in request")
		return err
	}
	return m.msgAPI.HandleRequest(req, replyTo)
}

// Start the digital twin module and open its native thing backup
// This subscribes to devices and agents that have a digital twin in the directory.
func (m *DigitwinModule) Start(_ string) (err error) {

	moduleID := m.GetModuleID()
	slog.Info("Start: Starting digitwin module", "moduleID", moduleID)

	// the vcache holds the cached notifications
	// if it doesn't contain a value it should forward the request to the device
	// note that the thingID is the digital twin ID, which needs to be converted
	// back to the device thingID
	m.vcache = vcache.NewVCacheModule()
	m.vcache.SetRequestSink(m.ForwardDigitwinRequestToDevice)
	m.vcache.Start("")
	// the device directory holds the unmodified device TDs
	m.deviceDirectory = directory.NewDirectoryServer(m.storageRoot, nil)
	m.deviceDirectory.Start("")

	storageDir := ""
	if m.storageRoot != "" {
		storageDir = filepath.Join(m.storageRoot, moduleID)
	}
	m.bucketStore, err = bucketstore.NewBucketStore(storageDir, bucketstoreapi.BackendKVBTree)

	err = m.bucketStore.Open()
	if err == nil {
		m.bucketName = moduleID
		m.bucket = m.bucketStore.GetBucket(m.bucketName)
	}
	// handling of messages for this module itself
	if err == nil {
		m.msgAPI = NewDigitwinMsgHandler(m)
	}

	m.directory.SetTDHooks(m.HandleWriteDirectory, m.HandleDeleteTD)

	// Subscribe to devices.
	// lets hope there aren't too many or this can take a while.
	// how to support wildcard device subscriptions? flatten the list of agents?
	// digitalTwins, err := m.directory.RetrieveAllThings(0, 0)

	// TODO: agents are subscribed to when they (re)connect,
	// so subscribe to server 'connect' notifications instead

	return nil
}

// Stop the digital twin module and release the allocation resources
func (m *DigitwinModule) Stop() {
	slog.Info("Stop: closing digitwin store")
	err := m.bucket.Close()
	if err != nil {
		slog.Error("Stop: error stopping digitwin bucket", "err", err.Error())
	}
	m.bucketStore.Close()
	m.vcache.Stop()
	m.deviceDirectory.Stop()
}

// Create a new digital twin module.
//
// storageRoot is the root directory where modules create their storage, "" for in-memory testing
//
// thingDir is the directory module that holds Thing TDs.
//
// addForms is a handler from a transport server for injecting forms in digital twin TDs
// that describe how to interact via the server's protocols. Each transport server
// provides a compatible handler.
func NewDigitwinModule(storageRoot string,
	thingDir directoryapi.IDirectoryServer,
	addforms func(tdoc *td.TD, includeAffordances bool)) *DigitwinModule {

	m := &DigitwinModule{
		addForms:               addforms,
		directory:              thingDir,
		storageRoot:            storageRoot,
		includeAffordanceForms: true,
	}
	m.SetModuleID(digitwinapi.DefaultDigitwinModuleID)

	var _ digitwinapi.IDigitwinModule = m // interface check
	return m
}

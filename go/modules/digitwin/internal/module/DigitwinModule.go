package module

import (
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/bucketstore"
	bucketstoreapi "github.com/hiveot/hivekit/go/modules/bucketstore/api"
	digitwinapi "github.com/hiveot/hivekit/go/modules/digitwin/api"
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

	// track the connections of agents
	agentStatus sync.Map

	// hook to server to add forms to a TD for interacting with affordances
	addForms func(tdoc *td.TD, includeAffordances bool)

	// internal storage with the original TDs
	bucket      bucketstoreapi.IBucket
	bucketStore bucketstoreapi.IBucketStore

	// the device directory holding TD's of the native devices/agents
	// deviceDirectory directoryapi.IDirectoryServer

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
	vcache vcacheapi.IVCacheServer

	// location of the digital twin storage location
	storageDir string
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

// func (m *DigitwinModule) GetDeviceDirectory() directoryapi.IDirectoryServer {
// 	return m.deviceDirectory
// }

// Return the unmarshalled device TD
// TODO: cache the unmarshalled TDs for faster handling
func (m *DigitwinModule) GetDeviceTD(thingID string) *td.TD {
	tdJson, err := m.bucket.Get(thingID)
	if err != nil {
		return nil
	}
	tdi, err := td.UnmarshalTD(string(tdJson))
	return tdi
}

// HandleNotification stores the latest notification things for retrieval as a digital twin value
// These notifications are received from (RC) agents connected to the server and from standalone devices.
// This also includes connection events from the server, which are used to update agent online status.
func (m *DigitwinModule) HandleNotification(notif *msg.NotificationMessage) {

	// track online status of agents - this needs tracking of agents
	// agentInfo := m.deviceDirectory.GetAgent(notif.SenderID)
	if notif.Name == transports.ConnectedEventName {
		// if this is an agent then its things are no longer online
		cinfo := transports.ConnectionInfo{}
		err := notif.Decode(&cinfo)
		if err == nil {
			m.SetAgentStatus(cinfo.ClientID, true)
		}
		// send notifications upstream to potential consumers
		m.ForwardNotification(notif)
		return
	} else if notif.Name == transports.DisconnectedEventName {
		// if this is an agent then its things are no longer online
		cinfo := transports.ConnectionInfo{}
		err := notif.Decode(&cinfo)
		if err == nil {
			m.SetAgentStatus(cinfo.ClientID, false)
		}
		// send notifications upstream to potential consumers
		m.ForwardNotification(notif)
		return
	}
	// if the thingID is a digital twin then store its value in the vcache
	dtwThingID := MakeDigitwinID(notif.SenderID, notif.ThingID)
	_, err := m.directory.RetrieveThing(dtwThingID)
	if err == nil {
		dtwNotif := *notif
		dtwNotif.ThingID = dtwThingID
		m.vcache.HandleNotification(&dtwNotif)
		// emit this notification as a digital twin update
		m.ForwardNotification(&dtwNotif)

	} else {
		// not a digital twin notification. Send it upstream to potential consumers
		m.ForwardNotification(notif)
	}

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
	// TODO: try to remove the thingID dependency on the digital twin.
	// Maybe lookup the thingID in the digital twin directory... ?
	if strings.HasPrefix(req.ThingID, digitwinapi.DigitwinIDPrefix) {
		switch req.Operation {

		// read requests are handled by the value cache
		// the value cache holds digital twin property and event notifications

		// FIXME: unobservable properties will never be in the cache. If the value
		// isn't cached the request should be forwarded. Eg set vcache sink to the
		// server or client connection that can forward it.
		case wot.OpReadAllProperties,
			wot.OpReadMultipleProperties,
			wot.OpReadProperty, // this returns the property value
			wot.HTOpReadEvent,  // this returns the event notification (not just the value)
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

// Set the connected status of an agent
// TODO: This updates the online status of all its digitwin devices
func (m *DigitwinModule) SetAgentStatus(agentID string, connected bool) {
	// if this is an agent then its things are no longer online
	m.agentStatus.Store(agentID, connected)
	// IDList := m.directory.GetThingsByAgentID(agentID)
	// for _, thingID := range IDList {
	// m.vcache.SetProperty(thingID, digitwinapi.OnlinePropName, connected)
	// // vcache will notify subscribers
	// }
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

	storageDir := ""
	m.bucketStore, err = bucketstore.NewBucketStore(storageDir, bucketstoreapi.BackendKVBTree)

	err = m.bucketStore.Open()
	if err == nil {
		m.bucket = m.bucketStore.GetBucket(moduleID)
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
	// m.deviceDirectory.Stop()
}

// NewDigitwinModule creates a new digital twin module instance.
//
//	storageDir is the directory where the module stores its data, "" for in-memory testing
//	thingDir is the directory module that holds Thing TDs.
//	addForms is a handler from a transport server for injecting forms in digital twin TDs
//	that describe how to interact via the server's protocols. Each transport server
//	provides a compatible handler.
func NewDigitwinModule(storageDir string,
	thingDir directoryapi.IDirectoryServer,
	addforms func(tdoc *td.TD, includeAffordances bool)) *DigitwinModule {

	m := &DigitwinModule{
		addForms:               addforms,
		directory:              thingDir,
		storageDir:             storageDir,
		includeAffordanceForms: true,
	}
	m.SetModuleID(digitwinapi.DefaultDigitwinModuleID)

	var _ digitwinapi.IDigitwinModule = m // interface check
	return m
}

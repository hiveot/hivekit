package internal

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/bucketstore"
	bucketstorepkg "github.com/hiveot/hivekit/go/modules/bucketstore/pkg"
	"github.com/hiveot/hivekit/go/modules/digitwin"
	"github.com/hiveot/hivekit/go/modules/directory"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/modules/vcache"
	vcacheapi "github.com/hiveot/hivekit/go/modules/vcache/api"
)

// DefaultDigitwinServiceID is the default moduleID of the digital twin module instance.
const DefaultDigitwinServiceID = "digitwin"

// DigitwinService implements the digital twin service module.
//
// This module serves a Digital Twin for eligible devices. It hooks into the provided
// thing directory to replace the device TD with a digital twin.
// It subscribes to notifications from registered devices and handles read and write requests for the
// digital twin. Where neccesary it forwards requests to the actual device if reachable.
//
// This tracks the TDs of the original devices in a separate device directory for use
// by modules like the router.
type DigitwinService struct {
	modules.HiveModuleBase

	// track the connections of agents
	agentStatus sync.Map

	// hook to server to add secforms to a TD for interacting with affordances
	addForms func(tdoc *td.TD, includeAffordances bool)

	// internal storage with the original device TDs
	deviceTDBucket bucketstore.IBucket
	deviceTDStore  bucketstore.IBucketStorage

	// the device directory holding TD's of the native devices/agents
	// deviceDirectory directory.IDirectoryServer

	// the Thing directory with digital twin TDs
	// this also contains TDs of non-digital twin devices and services, as consumers
	// should be able to use these as well.
	directory directory.IDirectoryService

	// The digitwin service instance thing-ID for handling requests
	digitwinThingID string

	// the store that holds the digital twin TDs and value
	digitwinStore bucketstore.IBucketStorage

	// configuration to add forms to all the affordances of a TD
	includeAffordanceForms bool

	// the RRN messaging API for the digitwin module itself
	msgAPI *DigitwinMsgHandler

	// notification cache holding device property and events values
	vcache vcacheapi.IVCacheService

	// location of the digital twin storage location
	storageDir string
}

// ForwardDigitalTwinRequest passes the request made to a digital twin to the original device.
// This will restore the original device thingID before forwarding the request.
func (m *DigitwinService) ForwardDigitwinRequestToDevice(dtwReq *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
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
		return replyTo(resp)
	})
	return err
}

// func (m *DigitwinModule) GetDeviceDirectory() directory.IDirectoryServer {
// 	return m.deviceDirectory
// }

// Return the unmarshalled device TD
// TODO: cache the unmarshalled TDs for faster handling
func (m *DigitwinService) GetDeviceTD(thingID string) *td.TD {
	tdJson, err := m.deviceTDBucket.Get(thingID)
	if err != nil {
		return nil
	}
	tdi, err := td.UnmarshalTD(string(tdJson))
	return tdi
}

// HandleNotification stores the latest notification things for retrieval as a digital twin value
// These notifications are received from (RC) agents connected to the server and from standalone devices.
// This also includes connection events from the server, which are used to update agent online status.
func (m *DigitwinService) HandleNotification(notif *msg.NotificationMessage) {

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
func (m *DigitwinService) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {

	// Handle requests for a digital twin
	// TODO: try to remove the thingID dependency on the digital twin.
	// Maybe lookup the thingID in the digital twin directory... ?
	if strings.HasPrefix(req.ThingID, digitwin.DigitwinIDPrefix) {
		switch req.Operation {

		// read requests are handled by the value cache
		// the value cache holds digital twin property and event notifications

		// Note unobservable properties will never be in the cache. If the value isn't cached
		// the request is forwarded to the device. Eg set vcache sink to the server or client
		// connection that can forward it.
		case td.OpReadAllProperties,
			td.OpReadMultipleProperties,
			td.OpReadProperty, // this returns the property value
			td.HTOpReadEvent,  // this returns the event notification (not just the value)
			td.HTOpReadAllEvents:
			return m.vcache.HandleRequest(req, replyTo)

		// write requests are forwarded to the actual device after mapping
		// the thingID back to that of the device
		case td.OpWriteProperty,
			td.OpWriteMultipleProperties,
			td.OpInvokeAction:

			return m.ForwardDigitwinRequestToDevice(req, replyTo)
		}
	}

	// Handle requests for this module
	if req.ThingID != m.digitwinThingID {
		return nil
	} else if req.SenderID == "" {
		err := fmt.Errorf("missing senderID in request")
		return err
	}
	return m.msgAPI.HandleRequest(req, replyTo)
}

// Set the connected status of an agent
// TODO: This updates the online status of all its digitwin devices
func (m *DigitwinService) SetAgentStatus(agentID string, connected bool) {
	// if this is an agent then its things are no longer online
	m.agentStatus.Store(agentID, connected)
	// IDList := m.directory.GetThingsByAgentID(agentID)
	// for _, thingID := range IDList {
	// m.vcache.SetProperty(thingID, digitwin.OnlinePropName, connected)
	// // vcache will notify subscribers
	// }
}

// Start the digital twin module and open its native thing backup
// This subscribes to devices and agents that have a digital twin in the directory.
func (m *DigitwinService) Start() (err error) {

	slog.Info("Start: Starting digitwin module")

	// the vcache holds the cached notifications
	// if it doesn't contain a value it should forward the request to the device
	// note that the thingID is the digital twin ID, which needs to be converted
	// back to the device thingID
	m.vcache = vcache.NewVCacheService()
	m.vcache.SetRequestSink(m.ForwardDigitwinRequestToDevice)
	m.vcache.Start()
	storageFile := filepath.Join(m.storageDir, "deviceTD.kvbtree")
	m.deviceTDStore, err = bucketstorepkg.NewBucketStore(storageFile, bucketstore.BackendKVBTree)

	err = m.deviceTDStore.Open()
	if err == nil {
		m.deviceTDBucket = m.deviceTDStore.GetBucket(m.digitwinThingID)
	}
	// handling of messages for this module itself
	if err == nil {
		m.msgAPI = NewDigitwinMsgHandler(m)
	}

	m.directory.SetTDHooks(m.HandleWriteDirectory, m.HandleDeleteTD)

	// FIXME: Subscribe to devices.
	// lets hope there aren't too many or this can take a while.
	// how to support wildcard device subscriptions? flatten the list of agents?
	// digitalTwins, err := m.directory.RetrieveAllThings(0, 0)

	// FIXME: agents are subscribed to when they (re)connect,
	// so subscribe to server 'connect' notifications instead

	return nil
}

// Stop the digital twin module and release the allocation resources
func (m *DigitwinService) Stop() {
	slog.Info("Stop: stopping digitwin module")
	err := m.deviceTDBucket.Close()
	if err != nil {
		slog.Error("Stop: error stopping digitwin bucket", "err", err.Error())
	}
	m.deviceTDStore.Close()
	m.vcache.Stop()
	// m.deviceDirectory.Stop()
}

// NewDigitwinService creates a new digital twin module instance.
//
//	storageDir is the location where the module stores its device TDs, "" for in-memory testing
//	thingDir is the directory module that holds Thing TDs.
//	addForms is a handler from a transport server for injecting forms in digital twin TDs
//	that describe how to interact via the server protocols. Each transport server
//	provides a compatible handler.
func NewDigitwinService(storageDir string,
	thingDir directory.IDirectoryService,
	addforms func(tdoc *td.TD, includeAffordances bool)) *DigitwinService {

	m := &DigitwinService{
		addForms:               addforms,
		digitwinThingID:        digitwin.DefaultDigitwinThingID,
		directory:              thingDir,
		storageDir:             storageDir,
		includeAffordanceForms: true,
	}

	var _ digitwin.IDigitwinService = m // interface check
	return m
}

package internal

import (
	"log/slog"
	"path/filepath"
	"sync"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/cells"
	"github.com/hiveot/hivekit/go/cells/bucketstore"
	"github.com/hiveot/hivekit/go/cells/bucketstore/kvbtreestore"
	"github.com/hiveot/hivekit/go/cells/digitwin"
	"github.com/hiveot/hivekit/go/cells/directory"
	"github.com/hiveot/hivekit/go/cells/vcache"
	vcache_service "github.com/hiveot/hivekit/go/cells/vcache/service"
)

// DefaultDigitwinServiceID is the default ID of the digital twin service instance.
const DefaultDigitwinServiceID = "digitwin"

// DigitwinServiceImpl implements the digital twin service.
//
// This service serves a Digital Twin for eligible devices. It hooks into the provided thing directory
// to replace the device TD with a digital twin.
//
// This:
//  1. Intercepts writing of TDs to the directory and injects the digital twin TD instead.
//     The original the device TD is stored in an internal hidden directory for use to forward requests.
//  2. Handles requests directed at the digital twin.
//     a. Handles read requests from the latest cached value.
//     b. Forwards action and property write requests to the device.
//  3. Auto-connect to the original device when a request for a digital twin is received.
//     the downstream router manages these connections.
//     This supports connections to network devices and from devices that use RC (reverse connections)
//
// For this to work the request and notification chains must be setup as follows:
//
// request chain:
//  1. consumer -> server -> ... -> digitwin ->
//     2A. router -> device clients -> device
//     2B. router -> server (RC device/service)
//
// notification chain:
//  3. device -> device clients -> router -> digitwin
//  4. rc device/service -> server -> router -> digitwin
//  5. digitwin -> ... -> server -> consumer
//
// Both the digitwin test package and the digitwin example in the examples section shows a how to establish
// this chain.
//
// Explanation:
//
//  1. Consumers connect to the digitwin service instead of the IoT device, because the TD of the
//     IoT device is modified to point to the digitwin service.
//     The transport server passes requests from consumers via middleware to the digital twin service.
//     the digital twin handles the request depending on the request type:
//     * property read requests are answered from the value cache if the property is observable.
//     * event read requests (a hiveot extension) are also answered from the value cache.
//     * action query requests are also answered from the value cache.
//     * all other requests are modified to use the original device ThingID and passed to the router (2).
//     * if the TD is not a digital twin TD, then the request is also forwarded to the router.
//
//  2. The router looks-up the Thing connection information from the provided thing directory.
//     This uses the internal directory where the digital twin stored the original Thing TD.
//
//     2A. If the Thing is a network device then the router connects if needed and forwards the request
//     to the device. Device connections are cached by the router.
//
//     2B. If the Thing is served by a RC device then the request is forwarded to the
//     transport server which locates the device connection using the device clientID
//     and forwards the request.
//     TDs provided by RC devices all include the clientID in the TD instead of forms.
//     WoT doesn't support/specify RC so this is the HiveOT specific solution.
//
//  3. Notifications received from devices are send down the device connection that was established
//     by the router. The router passes it upstream to the digitwin service.
//
//  4. Notifications received via RC arrive at the server which pass it to the router
//     which forwards it to the digitwin service.
//
//  5. The digitwin receives the notifications from the router.
//     If the ThingID has a digital twin then the value cache of the digital twin is updated.
//     Updating the digital twin value cache results in a new notification to be forwarded upstream.
//     this new notification is forwarded to the server which passes it to subscribers.
type DigitwinServiceImpl struct {
	*cells.HiveCellBase

	// track the connections of RC devices
	deviceStatus sync.Map

	// hook to server to add secforms to a TD for interacting with affordances
	addForms func(tdoc *td.TD, includeAffordances bool)

	// internal storage with the original hidden device TDs
	deviceTDBucket bucketstore.IBucket
	deviceTDStore  bucketstore.IBucketStore

	// the device directory holding TD's of the native devices.
	// deviceDirectory directory.IDirectoryServer

	// The application Thing directory where digital twin TDs are stored.
	// This also contains TDs of non-digital twin devices, as consumers
	// should be able to use these as well.
	directory directory.IDirectoryService

	// The digitwin service instance thing-ID for handling requests
	// digitwinThingID string

	// the store that holds the digital twin TDs and value
	digitwinStore bucketstore.IBucketStore

	// configuration to add forms to all the affordances of a TD
	includeAffordanceForms bool

	// notification cache holding device property and events values
	vcache vcache.IValueCacheService

	// location of the digital twin storage location
	storageDir string
}

// EmitDigitalTwinRequest sends request made to a digital twin to the original device.
// This will restore the original thingID before forwarding the request to
// the device itself.
func (svc *DigitwinServiceImpl) EmitDigitwinRequestToDevice(dtwReq *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
	// reverse the digital twin thingID
	clientID, thingID, err := SplitDigitwinID(dtwReq.ThingID)

	// the device itself expects the actual thingID, identifying itself or one of
	// its nested devices.
	deviceReq := *dtwReq
	deviceReq.ThingID = thingID

	// Emit the request to the sink, which is responsible for routing it to the
	// destination.
	_ = clientID
	err = svc.EmitRequest(&deviceReq, func(resp *msg.ResponseMessage) error {
		// put the digitwin thingID back into the response
		resp.ThingID = dtwReq.ThingID
		return replyTo(resp)
	})
	return err
}

// func (m *DigitwinServiceImple) GetDeviceDirectory() directory.IDirectoryServer {
// 	return m.deviceDirectory
// }

// Return the unmarshalled device TD
// TODO: cache the unmarshalled TDs for faster handling
func (svc *DigitwinServiceImpl) GetDeviceTD(thingID string) *td.TD {
	tdJson, err := svc.deviceTDBucket.Get(thingID)
	if err != nil {
		return nil
	}
	tdi, err := td.UnmarshalTD(string(tdJson))
	return tdi
}

// HandleNotification stores the latest notification things for retrieval as a digital twin value
// These notifications are passed from upstream, eg the router that connects to devices, or from
// downstream when the server that receives RC client notifications pushes out an event.
//
// This requires that the digitwin service is set as the notification sink from both the
// router and the servers, as both can receive events from remote Things.
//
// This also includes connection events from the server, which are used to update client online status.
func (svc *DigitwinServiceImpl) HandleNotification(notif *msg.NotificationMessage) {

	// track online status of devices - this needs tracking of devices
	// FIXME: how to know if senderID is a device?
	if notif.Name == api.ServerConnectedEvent {

		// if the sender is an device or service instead of a consumer then its things are now online.
		cinfo := api.ConnectionInfo{}
		err := notif.Decode(&cinfo)
		if err == nil {
			svc.SetDeviceStatus(cinfo.ClientID, true)
		}
		// send notifications upstream to potential consumers
		svc.ForwardNotification(notif)
		return
	} else if notif.Name == api.ServerDisconnectedEvent {
		// if this is an device or service then its things are no longer online
		cinfo := api.ConnectionInfo{}
		err := notif.Decode(&cinfo)
		if err == nil {
			svc.SetDeviceStatus(cinfo.ClientID, false)
		}
		// send notifications upstream to potential consumers
		svc.ForwardNotification(notif)
		return
	}

	// If the thingID is a digital twin then store its value in the vcache
	// TODO: can the dependency on SenderID be removed?
	dtwThingID := MakeDigitwinID(notif.SenderID, notif.ThingID)
	_, err := svc.directory.RetrieveThing(dtwThingID)
	if err == nil {
		dtwNotif := *notif
		dtwNotif.ThingID = dtwThingID
		svc.vcache.HandleNotification(&dtwNotif)
		// emit this notification as a digital twin update
		svc.EmitNotification(&dtwNotif)

	} else {
		// not a digital twin notification. Forward it upstream to potential consumers
		svc.ForwardNotification(notif)
	}

}

// Set the connected status of a device
// TODO: update the online status of all its digitwin devices
func (svc *DigitwinServiceImpl) SetDeviceStatus(clientID string, connected bool) {

	svc.deviceStatus.Store(clientID, connected)

}

// Start the digital twin service and open its native thing backup
// This subscribes to devices that have a digital twin in the directory.
func (svc *DigitwinServiceImpl) Start() (err error) {

	slog.Info("Start: Starting digitwin service")

	// the vcache holds the cached notifications
	// if it doesn't contain a value it should forward the request to the device
	// note that the thingID is the digital twin ID, which needs to be converted
	// back to the device thingID
	svc.vcache = vcache_service.NewValueCacheService()
	// don't set a sink
	// m.vcache.SetRequestSink(m.ForwardDigitwinRequestToDevice)
	svc.vcache.Start()
	storageFile := filepath.Join(svc.storageDir, "deviceTD.kvbtree")
	svc.deviceTDStore = kvbtreestore.NewBucketStore(storageFile)

	err = svc.deviceTDStore.Open()
	if err == nil {
		thingID := svc.GetThingID()
		svc.deviceTDBucket = svc.deviceTDStore.GetBucket(thingID)
	}

	svc.directory.SetTDHooks(svc.HandleWriteDirectory, svc.HandleDeleteTD)

	// FIXME: Subscribe to devices.
	// lets hope there aren't too many or this can take a while.
	// how to support wildcard device subscriptions? flatten the list of devices?
	// digitalTwins, err := m.directory.RetrieveAllThings(0, 0)

	// FIXME: RC clients are subscribed to when they (re)connect,
	// so subscribe to server 'connect' notifications.

	return nil
}

// Stop the digital twin service and release the allocation resources
func (svc *DigitwinServiceImpl) Stop() {
	slog.Info("Stop: stopping digitwin service")
	err := svc.deviceTDBucket.Close()
	if err != nil {
		slog.Error("Stop: error stopping digitwin bucket", "err", err.Error())
	}
	svc.deviceTDStore.Close()
	svc.vcache.Stop()
	// m.deviceDirectory.Stop()
}

// NewDigitwinServiceImpl creates a new digital twin service instance.
//
// This cells uses a directory to store the digital twin TD's and has an internal store
// for hidden non-digitwin TDs used to pass requests to the actual devices.
//
//	storageDir is the location where the service stores original TDs, "" for in-memory testing.
//	thingDir is the directory service that holds exposed Thing TDs.
//	addForms is a handler from a transport server for injecting forms in digital twin TDs
//	that describe how to interact via the server protocols.
func NewDigitwinServiceImpl(storageDir string,
	thingDir directory.IDirectoryService,
	addforms func(tdoc *td.TD, includeAffordances bool)) *DigitwinServiceImpl {

	thingID := digitwin.DefaultDigitwinServiceID
	svc := &DigitwinServiceImpl{
		HiveCellBase:           cells.NewHiveCellBase(thingID, 0),
		addForms:               addforms,
		directory:              thingDir,
		storageDir:             storageDir,
		includeAffordanceForms: true,
	}

	var _ digitwin.IDigitwinService = svc // interface check
	return svc
}

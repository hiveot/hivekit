package thing

import (
	"fmt"
	"log/slog"
	"maps"
	"sync"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/directory"
	"github.com/hiveot/hivekit/go/modules/factory"
	"github.com/hiveot/hivekit/go/utils"
)

const ExposedThingModuleType = "exposed-thing"

// ExposedThing is a module representing an Exposed Thing for IoT device operations using the
// standard RRN (request-response-notification) messages. The RRN interface is
// compatible with all HiveKit modules.
//
// This module is intended to help building a 'ExposedThing' by:
//   - track ExposedThing status with SetState and GetState
//   - methods for publishing property updates, events, action status and TDs
//     automatic update of property, event and action state when using publish methods
//   - handle read requests for property, event and action status
//   - hook for handling requests directed at the ExposedThing
//
// Usage:
//  1. Set this module as the request sink of a transport connection so it can receive requests
//  2. Set this module notification sink to the transport connection so it can publish notifications
//  3. Set this module request sink to other modules that handle server side requests.
//
// Therefore if no appRequestHandler is set, then do not set the request sink to
// the connection for use to send requests.
type ExposedThing struct {
	*modules.HiveModuleBase

	// appRequestHook is the application handler of requests addressed to this module.
	//
	// HandleRequest will invoke this callback or forward requests not destined for
	// this module (moduleID != request.ThingID) to requestSink.
	appRequestHook msg.RequestHandler

	mux sync.RWMutex

	// Map of the nested Things managed by this module
	tstates map[string]*ThingState
}

// Return the state of a thing that is managed by this module.
// This module is a Thing and can also manage nested things, like a 1-ware hardware
// gateway managing 1-wire devices.
//
// thingID is the ID of a nested Thing or "" for the module's Thing itself.
//
// If no entry for thingID yet exists, one is created.
func (m *ExposedThing) GetState(thingID string) *ThingState {
	if thingID == "" {
		thingID = m.GetThingID()
	}
	m.mux.RLock()
	state, ok := m.tstates[thingID]
	m.mux.RUnlock()
	if !ok {
		m.mux.Lock()
		state = NewThingState(thingID)
		m.tstates[thingID] = state
		defer m.mux.Unlock()
	}
	return state
}

// HandleReadRequests handles reading of actions, events, and properties for a nested thing.
// This returns nil if the request was handled or an error if this is not a valid read request
// or the thingID is unknown.
func (m *ExposedThing) HandleReadRequests(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
	var found bool
	var output any

	m.mux.RLock()
	defer m.mux.RUnlock()
	state, ok := m.tstates[req.ThingID]
	if !ok {
		// not handled
		err = fmt.Errorf("Unknown thingID '%s' for module '%s'", req.ThingID, m.GetThingID())
		return err
	}

	switch req.Operation {

	case td.HTOpReadAllEvents:
		output = state.GetAllEvents()

	case td.OpReadAllProperties:
		output = state.GetAllProperties()

	case td.HTOpReadEvent:
		output, found = state.events[req.Name]
		if !found {
			err = fmt.Errorf("Unknown event '%s'", req.Name)
		}

	case td.OpReadProperty:
		val, found := state.properties[req.Name]
		output = val
		if !found {
			err = fmt.Errorf("Unknown property '%s'", req.Name)
		}

	case td.OpReadMultipleProperties:

		var keys []string
		err = req.Decode(&keys)
		if err != nil {
			err = fmt.Errorf("Invalid input: %w", err)
			break
		}
		props := make(map[string]any)
		for _, k := range keys {
			v, ok := state.properties[k]
			if ok {
				props[k] = v
			} else {
				// fail or ignore invalid key? -> graceful degradation
			}
		}
		output = props
	case td.OpQueryAction:
		actionResp, ok := state.actionResponse[req.Name]
		if ok {
			err = fmt.Errorf("Unknown action: %s", req.Name)
		}
		output = actionResp
	case td.OpQueryAllActions:
		output = maps.Clone(state.actionResponse)
	default:
		// not handled
		err = fmt.Errorf("Unhandled operation '%s'", req.Operation)
		return err
	}
	resp := req.CreateResponse(output, err)
	err = replyTo(resp)
	return err
}

// HandleRequest handles request with thingID set to this moduleID.
//
// If a request hook is set then pass the request to the hook. If the hook does not handle the
// request then it MUST forward it using ForwardRequest.
//
// Applications can also embed this module and override HandleRequest to handle requests themselves.
//
// Modules that override HandleRequest should first handle the request itself and
// only hand it over to this base method when there is nothing for them to do. This method
// simply forwards the request if no request handler hook is set.
func (m *ExposedThing) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {

	// application can set a hook for handling all requests
	m.mux.RLock()
	handler := m.appRequestHook
	m.mux.RUnlock()

	// invoke registered hook
	if handler != nil {
		err = handler(req, replyTo)
		return err
	}
	if req.ThingID == m.GetThingID() {
		err = m.HandleReadRequests(req, replyTo)
	} else {
		err = m.ForwardRequest(req, replyTo)
	}
	return err
}

// PubActionProgress helper for things to send a 'running' ActionStatus notification
//
// This sends an ResponseMessage message with status of running.
func (m *ExposedThing) PubActionProgress(req msg.RequestMessage, value any) {
	status := &msg.ResponseMessage{
		Name:      req.Name,
		Output:    value,
		SenderID:  m.GetThingID(),
		Status:    msg.StatusRunning,
		ThingID:   req.ThingID,
		Timestamp: utils.FormatNowUTCMilli(),
	}

	resp := msg.NewNotificationMessage(
		m.GetThingID(), msg.AffordanceTypeAction, req.ThingID, req.Name, status)

	m.GetState(req.ThingID).SetActionResponse(req.Name, status)

	m.ForwardNotification(resp)
}

// PubEvent helper for things to publish an event to the server.
//
//	thingID is the thing for which the module publishes the properties or "" for the module Thing itself.
//	name is the name of the event to publish.
//	value is the value of the event to publish, if any
func (m *ExposedThing) PubEvent(thingID string, name string, value any) {

	if thingID == "" {
		thingID = m.GetThingID()
	}

	// This is a response to subscription request.
	// for now assume this is a hub connection and the hub wants all events
	notif := msg.NewNotificationMessage(
		m.GetThingID(), msg.AffordanceTypeEvent, thingID, name, value)
	slog.Info("PubEvent",
		"thingID", thingID,
		"name", name,
		"value", notif.ToString(50),
	)
	m.GetState(thingID).SetEvent(name, notif)

	m.ForwardNotification(notif)
}

// PubProperty publishes a property change notification to observers,
// and store the notification in the state store.
//
// Do not publish non-observable properties like date/time and counters, unless intentional.
//
//	thingID is the thing for which the module publishes the properties or "" for the module itself
//	propName is the name of the property to publish.
//	propValue is the value of the property to publish.
//	onlyChanges flag only publish changed values.
func (m *ExposedThing) PubProperty(thingID string, propName string, propVal any, onlyChanges bool) {

	if thingID == "" {
		thingID = m.GetThingID()
	}
	tstate := m.GetState(thingID)
	hasChanged := true
	if onlyChanges {
		// since most values are native types a simple compare should suffice
		old, found := tstate.GetProperty(propName)
		if found && old == propVal {
			// if old != nil && reflect.DeepEqual(old, propVal) {
			hasChanged = false
		}
	}
	if hasChanged {
		// This is a response to an observation request.
		// send the property update as a response to the observe request
		notif := msg.NewNotificationMessage(
			m.GetThingID(), msg.AffordanceTypeProperty, thingID, propName, propVal)
		slog.Info("PubProperty",
			"thingID", thingID,
			"name", notif.Name,
			"value", notif.ToString(50),
		)
		tstate.SetProperty(propName, notif.Data)

		m.ForwardNotification(notif)
	}
}

// PubProperties publishes multiple property changes to observers
// This updates the Thing state map with the property values
//
//	thingID is the thing for which the module publishes the properties, or "" for the module itself
//	propMap is the map of properties to handle
//	onlyChanges flag only publish changed values
func (m *ExposedThing) PubProperties(thingID string, propMap map[string]any, onlyChanges bool) {
	if thingID == "" {
		thingID = m.GetThingID()
	}
	for propName, propVal := range propMap {
		m.PubProperty(thingID, propName, propVal, onlyChanges)
	}
}

// Set the hook to invoke when requests are received by this module.
//
// The handler is invoked when requests are received with the ThingID set to
// this Thing's moduleID.
//
// This hook is intended to implement Thing behavior without having to implement
// a separate module.
//
// The hook MUST either call replyTo with the result or return an error.
// Failure to do so results in the request being lost and the caller waiting
// for a response until timeout.
func (m *ExposedThing) SetAppRequestHook(hook msg.RequestHandler) {
	m.mux.Lock()
	defer m.mux.Unlock()
	m.appRequestHook = hook
}

// WriteTD publish a request downstream to write a TD to a directory or discovery service.
//
// This addresses the request to the DefaultDirectoryThingID. The directory service
// and the discovery service can both handle the request.
//
// If the application utilizes a reverse connection to a gateway. The request will
// be passed to the gateway where it is routed to the default directory. The TD
// can be a TM as the forms and auth info are not applicable.
//
// If the application runs its own server then it should place a discovery server module
// behind this module so it can publish the TD. The TD should contain the auth and form
// info for connecting to the server.
func (m *ExposedThing) WriteTD(tdJson string) error {

	err := m.Rpc(td.OpInvokeAction,
		directory.DefaultDirectoryThingID,
		directory.CreateThingAction,
		tdJson, nil)

	return err
}

// NewExposedThing creates a new exposed thing (producer) instance for serving requests and
// sending notifications.
//
//	thingID is the ID of the exposed Thing.
//	appReqHandler is the application handler invoked when receiving requests for this Thing.
func NewExposedThing(thingID string, appReqHandler msg.RequestHandler) *ExposedThing {

	m := &ExposedThing{
		// Things dont send requests so no wait
		HiveModuleBase: modules.NewHiveModuleBase(thingID, 0),
		tstates:        make(map[string]*ThingState),
	}

	if appReqHandler != nil {
		m.SetAppRequestHook(appReqHandler)
	}
	return m
}

// Factory for creating a thing module using the factory environment
func NewExposedThingFactory(f factory.IModuleFactory, md *factory.ModuleDefinition) (modules.IHiveModule, error) {
	appID := f.GetEnvironment().AppID
	c := NewExposedThing(appID, nil)
	return c, nil
}

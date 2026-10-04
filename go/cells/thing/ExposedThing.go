package thing

import (
	"fmt"
	"log/slog"
	"maps"
	"sync"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/cells"
	"github.com/hiveot/hivekit/go/cells/directory"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/teris-io/shortid"
)

const ExposedThingCellType = "exposed-thing"

// ExposedThing is a cell representing an Exposed Thing for IoT device operations using the
// standard RRN (request-response-notification) messages. The RRN interface is
// compatible with all HiveKit cells.
//
// This cell is intended to help building a 'ExposedThing' by:
//   - track ExposedThing status with SetState and GetState
//   - methods for publishing property updates, events, action status and TDs
//     automatic update of property, event and action state when using publish methods
//   - handle read requests for property, event and action status
//   - hook for handling requests directed at the ExposedThing
//
// Usage:
//  1. Set this cell as the request sink of a transport connection so it can receive requests
//  2. Set this cell notification sink to the transport connection so it can publish notifications
//  3. Set this cell request sink to other cells that handle server side requests.
//
// Therefore if no appRequestHandler is set, then do not set the request sink to
// the connection for use to send requests.
type ExposedThing struct {
	*cells.HiveCellBase

	// appRequestHook is the application handler of requests addressed to this cell.
	//
	// HandleRequest will invoke this callback or forward requests not destined for
	// this cell (cellID != request.ThingID) to requestSink.
	appRequestHook msg.RequestHandler

	mux sync.RWMutex

	// Map of the nested Things managed by this cell
	tstates map[string]*ThingState
}

// Return the state of a thing that is managed by this cell.
// This cell is a Thing and can also manage nested things, like a 1-ware hardware
// gateway managing 1-wire devices.
//
// thingID is the ID of a nested Thing or "" for the cell's Thing itself.
//
// If no entry for thingID yet exists, one is created.
func (svc *ExposedThing) GetState(thingID string) *ThingState {
	if thingID == "" {
		thingID = svc.GetID()
	}
	svc.mux.RLock()
	state, ok := svc.tstates[thingID]
	svc.mux.RUnlock()
	if !ok {
		svc.mux.Lock()
		state = NewThingState(thingID)
		svc.tstates[thingID] = state
		defer svc.mux.Unlock()
	}
	return state
}

// HandleReadRequests handles reading of actions, events, and properties for a nested thing.
// This returns nil if the request was handled or an error if this is not a valid read request
// or the thingID is unknown.
func (svc *ExposedThing) HandleReadRequests(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
	var found bool
	var output any

	svc.mux.RLock()
	defer svc.mux.RUnlock()
	state, ok := svc.tstates[req.ThingID]
	if !ok {
		// no properties or events have been recorded;
		// ething owner should use PubProperty or other method to create a tstate object.
		err = fmt.Errorf("No properties for thingID '%s'", req.ThingID)
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
		err = req.DecodeInput(&keys)
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

// HandleRequest handles exposed thing requests with thingID set to this cellID.
//
// If a request hook is set then pass the request to the hook. If the hook does not handle the
// request then it MUST forward it using ForwardRequest.
//
// This handles all property and event read operations and serves values that
// were published with PubProperty and PubEvent. Non-observable properties
// can be included by using SetProperty instead of PubProperty.
//
// Applications can also embed this cell and override HandleRequest to handle requests themselves.
//
// Cells that override HandleRequest should first handle the request itself and
// only hand it over to this base method when there is nothing for them to do. This method
// simply forwards the request if no request handler hook is set.
func (svc *ExposedThing) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {

	// application can set a hook for handling all requests
	svc.mux.RLock()
	handler := svc.appRequestHook
	svc.mux.RUnlock()

	// invoke registered hook
	if handler != nil {
		err = handler(req, replyTo)
		return err
	}
	if req.ThingID == svc.GetID() {
		err = svc.HandleReadRequests(req, replyTo)
	} else {
		err = svc.ForwardRequest(req, replyTo)
	}
	return err
}

// PubActionProgress helper for things to send a 'running' ActionStatus notification
//
// This sends an ResponseMessage message with status of running.
func (svc *ExposedThing) PubActionProgress(req msg.RequestMessage, value any) {
	status := &msg.ResponseMessage{
		Name:      req.Name,
		Output:    value,
		SenderID:  svc.GetID(),
		Status:    msg.StatusRunning,
		ThingID:   req.ThingID,
		Timestamp: utils.FormatNowUTCMilli(),
	}

	resp := msg.NewNotificationMessage(
		svc.GetID(), msg.AffordanceTypeAction, req.ThingID, req.Name, status)

	svc.GetState(req.ThingID).SetActionResponse(req.Name, status)

	svc.EmitNotification(resp)
}

// PubEvent helper for things to publish an event to the server.
//
//	thingID is the thing for which the cell publishes the properties or "" for the cell Thing itself.
//	name is the name of the event to publish.
//	value is the value of the event to publish, if any
func (svc *ExposedThing) PubEvent(thingID string, name string, value any) {

	if thingID == "" {
		thingID = svc.GetID()
	}

	// This is a response to subscription request.
	// for now assume this is a hub connection and the hub wants all events
	notif := msg.NewNotificationMessage(
		svc.GetID(), msg.AffordanceTypeEvent, thingID, name, value)
	slog.Info("PubEvent",
		"thingID", thingID,
		"name", name,
		"value", notif.ToString(50),
	)
	svc.GetState(thingID).SetEvent(name, notif)

	svc.EmitNotification(notif)
}

// PubProperty publishes a property change notification to observers,
// and store the notification in the state store.
//
// Do not publish non-observable properties like date/time and counters, unless intentional.
//
//	thingID is the thing for which the cell publishes the properties or "" for the cell itself
//	propName is the name of the property to publish.
//	propValue is the value of the property to publish.
//	onlyChanges flag only publish changed values.
func (svc *ExposedThing) PubProperty(thingID string, propName string, propVal any, onlyChanges bool) {

	if thingID == "" {
		thingID = svc.GetID()
	}
	tstate := svc.GetState(thingID)
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
			svc.GetID(), msg.AffordanceTypeProperty, thingID, propName, propVal)
		slog.Info("PubProperty",
			"thingID", thingID,
			"name", notif.Name,
			"value", notif.ToString(50),
		)
		tstate.SetProperty(propName, notif.Data)

		svc.EmitNotification(notif)
	}
}

// PubProperties publishes multiple property changes to observers
// This updates the Thing state map with the property values
//
//	thingID is the thing for which the cell publishes the properties, or "" for the cell itself
//	propMap is the map of properties to handle
//	onlyChanges flag only publish changed values
func (svc *ExposedThing) PubProperties(thingID string, propMap map[string]any, onlyChanges bool) {
	if thingID == "" {
		thingID = svc.GetID()
	}
	for propName, propVal := range propMap {
		svc.PubProperty(thingID, propName, propVal, onlyChanges)
	}
}

// Publish the exposed thing's TD to the directory.
// This sends the directory UpdateTD request message to the cell request sink.
//
//	tdJSON is the TD to write.
func (svc *ExposedThing) PublishTD(tdJSON string) error {
	reqSink := svc.GetRequestSink()
	if reqSink == nil {
		return fmt.Errorf("PublishTD: No request sink set.")
	}
	// FIXME: how to get the directory TD or thingID?.
	//  Only needed if no connection exists and a router is present.
	directoryThingID := ""
	err := svc.Rpc(td.OpInvokeAction, directoryThingID, directory.UpdateThingAction, tdJSON, nil)
	return err
}

// Set the hook to invoke when requests are received by this cell.
//
// The handler is invoked when requests are received with the ThingID set to
// this Thing's cellID.
//
// This hook is intended to implement Thing behavior without having to implement
// a separate cell.
//
// The hook MUST either call replyTo with the result or return an error.
// Failure to do so results in the request being lost and the caller waiting
// for a response until timeout.
func (svc *ExposedThing) SetAppRequestHook(hook msg.RequestHandler) {
	svc.mux.Lock()
	defer svc.mux.Unlock()
	svc.appRequestHook = hook
}

// SetProperty updates the store property value so it can be read ReadProperty(ies).
//
// Intended for non-observable properties such as date/time and counters.
// Use PubProperty for observable properties instead.
//
//	thingID is the thing whose propery value to update or "" for the cell itself.
//	propName is the name of the property to set.
//	propValue is the value of the property to set.
//
// This returns a flag whether the property value has changed
func (svc *ExposedThing) SetProperty(thingID string, propName string, propVal any) (changed bool) {

	if thingID == "" {
		thingID = svc.GetID()
	}
	tstate := svc.GetState(thingID)

	// since most values are native types a simple compare should suffice
	hasChanged := true
	old, found := tstate.GetProperty(propName)
	if found && old == propVal {
		// if old != nil && reflect.DeepEqual(old, propVal) {
		hasChanged = false
	}
	tstate.SetProperty(propName, propVal)
	return hasChanged
}

// NewExposedThing creates a ready to use exposed thing (device or service) instance
// for serving requests and sending notifications.
//
// This handles publishing properties and events, tracks property values,
// and handle property read requests.
//
// A request hook allows an application to use the ExposedThing as a Thing
// and simply hook their request handler into it without embedding.
//
//	thingID is the ID of the exposed Thing.
//	appReqHandler is the application handler invoked when receiving requests for this Thing.
func NewExposedThing(thingID string, appReqHandler msg.RequestHandler) *ExposedThing {

	ething := &ExposedThing{
		// Things dont send requests so no wait
		HiveCellBase: cells.NewHiveCellBase(thingID),
		tstates:      make(map[string]*ThingState),
	}

	if appReqHandler != nil {
		ething.SetAppRequestHook(appReqHandler)
	}
	return ething
}

// Factory for creating an exposed Thing using the factory environment.
//
// This uses the Cell Type name as the thingID prefix followed by shortid.
func NewExposedThingFactory(f api.ICellFactory, def *api.CellDefinition) (api.IHiveCell, error) {
	thingID := def.Type + "-" + shortid.MustGenerate()
	c := NewExposedThing(thingID, nil)
	return c, nil
}

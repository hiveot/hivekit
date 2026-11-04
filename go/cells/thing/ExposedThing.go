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
	"github.com/teris-io/shortid"
)

const ExposedThingCellType = "exposed-thing"

// ExposedThing is a cell representing an Exposed Thing for IoT device operations using the
// standard RRN (request-response-notification) messages. The RRN interface is
// used by all HiveOT cells.
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

	// The TD describing the properties, events and actions used here
	// tdoc     *td.TD
	// tdocJSON string

	// Exposed Thing State with last known properties, events and actions of this thing
	state ExposedThingState
}

// Return the JSON with the TD of this exposed thing and (optionally) all the things
// that are managed by it.
// Intended for use by consumers/gateway that want to know what is available,
// espially from devices that manage nested Things like zwave, 1-wire, etc.
//
// List of TD's in JSON
// func (eThing *ExposedThing) GetAllThings() []string {
// 	return []string{eThing.tdocJSON}
// }

func (eThing *ExposedThing) GetProperty(name string) (val any, found bool) {
	return eThing.GetProperty(name)
}

// HandleReadRequests handles reading of actions, events, and properties for a nested thing.
//
// This returns nil if the request was handled or an error if this is not a valid read request
// or the thingID is unknown.
//
// Additional thingIDs can be supported by publishing their properties/events.
func (eThing *ExposedThing) HandleReadRequests(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
	var output any

	eThing.mux.RLock()
	defer eThing.mux.RUnlock()

	switch req.Operation {

	case td.HTOpReadAllEvents:
		output = eThing.state.GetAllEvents()

	case td.OpReadAllProperties:
		output = eThing.state.GetAllProperties()

	case td.HTOpReadEvent:
		output = eThing.state.GetEvent(req.Name)
		if output == nil {
			err = fmt.Errorf("Unknown event '%s'", req.Name)
		}

	case td.OpReadProperty:
		val := eThing.state.GetProperty(req.Name)
		if val == nil {
			err = fmt.Errorf("Unknown property '%s'", req.Name)
		} else {
			output = val.Data
		}

	case td.OpReadMultipleProperties:

		var keys []string
		err = req.DecodeInput(&keys)
		if err != nil {
			err = fmt.Errorf("Invalid input: %w", err)
			break
		}
		props := make(map[string]any)
		eThing.mux.RLock()
		for _, k := range keys {
			v, ok := eThing.state.properties[k]
			if ok {
				props[k] = v
			} else {
				// fail or ignore invalid key? -> graceful degradation
			}
		}
		eThing.mux.RUnlock()
		output = props
	case td.OpQueryAction:

		actionStatus := eThing.state.GetActionStatus(req.Name)
		if actionStatus == nil {
			err = fmt.Errorf("Unknown action: %s", req.Name)
		}
		output = actionStatus
	case td.OpQueryAllActions:
		output = maps.Clone(eThing.state.actions)

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
// request then the hook MUST forward the request using ForwardRequest.
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
func (eThing *ExposedThing) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {

	// application can set a hook for handling all requests
	eThing.mux.RLock()
	handler := eThing.appRequestHook
	eThing.mux.RUnlock()

	// invoke registered hook
	if handler != nil {
		err = handler(req, replyTo)
		return err
	}
	err = eThing.HandleReadRequests(req, replyTo)
	if err != nil {
		// anything other than read requests is not handled here
		err = eThing.ForwardRequest(req, replyTo)
	}
	return err
}

// PubActionProgress helper for things to send a 'running' ActionStatus notification
//
// This sends an Notification message with the action status
func (eThing *ExposedThing) PubActionStatus(thingID string, name string, status msg.ActionStatus, value any) {

	notif := msg.NewNotificationMessage(
		eThing.GetID(), msg.AffordanceTypeAction, thingID, name, status)

	eThing.EmitNotification(notif)
}

// PubEvent helper for things to publish an event value to the server.
//
//	thingID is the thing for which the cell publishes the properties or "" for the cell Thing itself.
//	name is the name of the event to publish.
//	value is the value of the event to publish, if any
func (eThing *ExposedThing) PubEvent(thingID string, name string, value any) {

	if thingID == "" {
		thingID = eThing.GetID()
	}

	// This is a response to subscription request.
	// for now assume this is a hub connection and the hub wants all events
	notif := msg.NewNotificationMessage(
		eThing.GetID(), msg.AffordanceTypeEvent, thingID, name, value)
	slog.Info("PubEvent",
		"thingID", thingID,
		"name", name,
		"value", notif.ToString(50),
	)
	eThing.state.SetEvent(name, notif)

	eThing.EmitNotification(notif)
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
func (eThing *ExposedThing) PubProperty(thingID string, propName string, propVal any, onlyChanges bool) {

	if thingID == "" {
		thingID = eThing.GetID()
	}
	hasChanged := true
	if onlyChanges {
		// since most values are native types a simple compare should suffice
		old := eThing.state.GetProperty(propName)
		if old != nil && old.Data == propVal {
			// if old != nil && reflect.DeepEqual(old, propVal) {
			hasChanged = false
		}
	}
	if hasChanged {
		// This is a response to an observation request.
		// send the property update as a response to the observe request
		notif := msg.NewNotificationMessage(
			eThing.GetID(), msg.AffordanceTypeProperty, thingID, propName, propVal)
		slog.Info("PubProperty",
			"thingID", thingID,
			"name", notif.Name,
			"value", notif.ToString(50),
		)
		eThing.state.SetProperty(propName, notif.Data)

		eThing.EmitNotification(notif)
	}
}

// PubProperties publishes multiple property changes to observers
// This updates the Thing state map with the property values
//
//	thingID is the thing for which the cell publishes the properties, or "" for the cell itself
//	propMap is the map of properties to handle
//	onlyChanges flag only publish changed values
func (eThing *ExposedThing) PubProperties(thingID string, propMap map[string]any, onlyChanges bool) {
	if thingID == "" {
		thingID = eThing.GetID()
	}
	for propName, propVal := range propMap {
		eThing.PubProperty(thingID, propName, propVal, onlyChanges)
	}
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
func (eThing *ExposedThing) SetAppRequestHook(hook msg.RequestHandler) {
	eThing.mux.Lock()
	defer eThing.mux.Unlock()
	eThing.appRequestHook = hook
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
func (eThing *ExposedThing) SetProperty(thingID string, propName string, propVal any) (changed bool) {

	if thingID == "" {
		thingID = eThing.GetID()
	}

	// since most values are native types a simple compare should suffice
	hasChanged := true
	old := eThing.state.GetProperty(propName)
	if old != nil && old.Data == propVal {
		// if old != nil && reflect.DeepEqual(old, propVal) {
		hasChanged = false
	}
	eThing.state.SetProperty(propName, propVal)
	return hasChanged
}

// Write the exposed thing's TD to the directory
//
// As per WoT recommendation, this sends an UpdateThing action request to the directory.
//
// Note that this requires that the linked transport sends the request to the
// directory server.
func (eThing *ExposedThing) WriteTD(tdoc *td.TD) error {

	slog.Info("WriteTD", "thingID", tdoc.ID)

	tdJSON := tdoc.ToJSON()

	// NOTE: if the directory thingID is unknown, leave it empty. If the connection exists then
	// the update request will be passed down the chain of cells to the directory, which recognizes
	// the request.
	// Only if the router needs to route it, must the thingID be known.
	directoryThingID := ""
	err := eThing.Rpc(td.OpInvokeAction, directoryThingID, directory.UpdateThingAction, tdJSON, nil)
	if err != nil {
		slog.Warn("PublishTD: Failed to update the directory", "err", err.Error())
	}
	return err
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
		state:        *NewThingState(),
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

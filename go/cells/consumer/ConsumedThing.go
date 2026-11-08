package consumer

import (
	"fmt"
	"sync"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/wot"
	"github.com/hiveot/hivekit/go/cells"
)

// A ConsumedThing represents a client interface to operate on a remote Thing.
//
// Modelled after ConsumedThing of the WoT scripting API:
//
//	https://w3c.github.io/wot-scripting-api/#dom-consumedthing
//
// This is work in progress. The intent is to track events and observe properties for local use.
// It is created by a 'consumer' that links the transport.
type ConsumedThing struct {
	cells.HiveCellBase

	// The TD of the thing represented here
	tdoc *wot.TD

	// property value
	propValues map[string]*msg.ThingValue

	// Thing event values
	// TBD: should this be an interaction output?
	// Updated when an event is published.
	eventValues map[string]*msg.ThingValue

	// Most recent action statuses
	actionStatuses map[string]*msg.ActionStatus

	// Consumers for sending requests. Optional
	co IConsumer

	// protect operations on values
	mux sync.RWMutex
}

// Return the action status of a thing.
// This returns nil if no action status is available.
func (ct *ConsumedThing) GetActionStatus(name string) *msg.ActionStatus {
	value, _ := ct.actionStatuses[name]
	return value
}

// GetDataSchema returns the property, event or action dataschema
func (ct *ConsumedThing) GetDataSchema(
	affType msg.AffordanceType, name string) (schema *wot.DataSchema, found bool) {

	switch affType {
	case msg.AffordanceTypeAction:
		aff, found := ct.tdoc.Actions[name]
		if !found {
			return nil, false
		}
		schema = aff.Output

	case msg.AffordanceTypeEvent:
		aff, found := ct.tdoc.Events[name]
		if !found {
			return nil, false
		}
		schema = aff.Data

	case msg.AffordanceTypeProperty:
		aff, found := ct.tdoc.Properties[name]
		if !found {
			return nil, false
		}
		schema = &aff.DataSchema

	default:
		return nil, false
	}
	return schema, true
}

// Return the event value of a thing.
// This returns nil if no value is available.
func (ct *ConsumedThing) GetEventValue(name string) *msg.ThingValue {
	value, _ := ct.eventValues[name]
	return value
}

// Return an interaction output for the given affordance type and name.
//
// The interaction output contains the affordance type and value. The value
// is obtained from the cached values of received notifications.
//
// This returns the interaction out and a flag if a value was found.
// iout is nil if no affordance exists with that name.
func (ct *ConsumedThing) GetIOutput(
	affType msg.AffordanceType, name string) *InteractionOutput {

	// FIXME: cache interaction output instances and replace them when
	// a value from a notification is received. (iout is are immutable)
	tv := ct.GetValue(affType, name)
	if tv == nil {
		return nil
	}
	iout := NewInteractionOutput(ct.tdoc, affType, name, tv.Raw, tv.Timestamp)
	return iout
}

// Return the property value of a thing
func (ct *ConsumedThing) GetPropValue(name string) *msg.ThingValue {
	value, _ := ct.propValues[name]
	return value
}

// Return the property, event or action status value of the consumed thing
//
// This returns nil if the affordance value isn't found. This doesn't mean
// the affordance is not valid, just that there is no value.
func (ct *ConsumedThing) GetValue(
	affType msg.AffordanceType, name string) (tv *msg.ThingValue) {

	switch affType {
	case msg.AffordanceTypeAction:
		// return the output value of actions
		as := ct.GetActionStatus(name)
		if as == nil {
			return nil
		}
		tv = &msg.ThingValue{Raw: as.Output, Timestamp: as.TimeUpdated}
	case msg.AffordanceTypeEvent:
		tv = ct.GetEventValue(name)
	case msg.AffordanceTypeProperty:
		tv = ct.GetPropValue(name)
	}
	schema, _ := ct.GetDataSchema(affType, name)
	if schema != nil {
		tv.Unit = schema.Unit
	}
	return tv
}

// Return the TD of this consumed thing
func (ct *ConsumedThing) GetTD() *wot.TD {
	return ct.tdoc
}

// HandleNotification captures the values
func (ct *ConsumedThing) HandleNotification(notif *msg.NotificationMessage) {
	ct.mux.Lock()
	defer ct.mux.Unlock()

	tv := &msg.ThingValue{
		Raw:       notif.Data,
		Timestamp: notif.Timestamp,
	}
	schema, _ := ct.GetDataSchema(notif.AffordanceType, notif.Name)
	if schema != nil {
		tv.Unit = schema.Unit
	}

	switch notif.AffordanceType {
	case msg.AffordanceTypeAction:
		as := msg.ActionStatus{}
		notif.Decode(&as)
		ct.actionStatuses[notif.Name] = &as
	case msg.AffordanceTypeEvent:
		ct.eventValues[notif.Name] = tv
	case msg.AffordanceTypeProperty:
		ct.propValues[notif.Name] = tv
	}
}

// Stop the consumed thing and unsubscribe events and properties
func (ct *ConsumedThing) Stop() {
	if ct.co != nil {
		ct.co.Unsubscribe(ct.tdoc.ID, "")
		ct.co.UnobserveProperty(ct.tdoc.ID, "")
	}
}

// Return a new consumed thing.
//
// This is similar to the consume() method in the WoT scripting api.
//
// The consumer handles the subscriptions and forwards requests related to actions,
// properties and events. Call Stop() to remove all subscriptions.
//
// If co is nil then this is just a storage container for Thing values.
//
// The co is used for sending requests and receiving events. nil to ignore.
// This subscribes to notification for the Thing.
func NewConsumedThing(tdoc *wot.TD, co IConsumer) (*ConsumedThing, error) {
	ct := &ConsumedThing{
		HiveCellBase: *cells.NewHiveCellBase(tdoc.ID),
		co:           co,
		tdoc:         tdoc,
	}
	if co != nil {
		co.Subscribe(tdoc.ID, "")
		co.ObserveProperty(tdoc.ID, "")

		ct.SetRequestSink(co)
		co.SetNotificationSink(ct, tdoc.ID)
	}

	// return ct
	return ct, fmt.Errorf("not yet implemented")
}

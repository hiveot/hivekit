package counterdevice

import (
	_ "embed"
	"fmt"
	"sync/atomic"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules"
	clientspkg "github.com/hiveot/hivekit/go/modules/clients/pkg"
	"github.com/hiveot/hivekit/go/modules/factory"
)

//go:embed "my-counter-tm.json"
var CounterDeviceTM []byte

// Module type for use in the recipe
const CounterDeviceModuleType = "counter-device"

// thingID requests are directed to
const DefaultCounterDeviceThingID = "counterThingID"

// Affordance IDs
const (
	CounterPropName     = "counter"
	CounterUpdatedEvent = "counterUpdated"
	DecrementActionName = "decrement"
	IncrementActionName = "increment"
)

// Simple IoT device that tracks a counter.
// The device uses Agent as a base. Agents facilitate storing and querying properties so you dont have to.
//
// This implements the properties, events and actions listed in the device TM.
// This does not expose the TM because .. this is a simple example.
type MyCounterDevice struct {
	clientspkg.Agent

	counter atomic.Int32
}

func (m *MyCounterDevice) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
	if req.ThingID != DefaultCounterDeviceThingID {
		return m.ForwardRequest(req, replyTo)
	}
	// use Agent to handle read properties/events/action requests
	err = m.HandleReadRequests(req, replyTo)
	if err == nil {
		return nil
	}

	// request was unhandled
	switch req.Operation {
	case td.OpInvokeAction:
		switch req.Name {
		case DecrementActionName:
			return m.HandleDecrement(req, replyTo)
		case IncrementActionName:
			return m.HandleIncrement(req, replyTo)
		}
	case td.OpWriteProperty:
		return m.HandleWriteProperty(req, replyTo)
	default:
		err = fmt.Errorf("Unhandled operation '%s'", req.Operation)
	}
	return err
}

func (m *MyCounterDevice) HandleDecrement(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
	m.counter.Add(-1)
	resp := req.CreateResponse(nil, nil)
	err = replyTo(resp)
	// PubEvent makes the last event available via HandleReadRequests
	m.PubEvent(req.ThingID, CounterUpdatedEvent, m.counter.Load())
	return err
}
func (m *MyCounterDevice) HandleIncrement(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
	m.counter.Add(1)
	resp := req.CreateResponse(nil, nil)
	err = replyTo(resp)
	// PubEvent makes the last event available via HandleReadRequests
	m.PubEvent(req.ThingID, CounterUpdatedEvent, m.counter.Load())
	return err
}
func (m *MyCounterDevice) HandleWriteProperty(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
	var newValue int

	err = req.Decode(&newValue)
	if err == nil {
		m.counter.Store(int32(newValue))
		// PubProperty makes the last value available via HandleReadRequests
		m.PubProperty(req.ThingID, req.Name, newValue)
	}
	resp := req.CreateResponse(nil, err)
	err = replyTo(resp)

	if err == nil {
		// PubEvent makes the last event available via HandleReadRequests
		m.PubEvent(req.ThingID, CounterUpdatedEvent, m.counter.Load())
	}
	return err
}

// Start the device module.
func (m *MyCounterDevice) Start() error {
	return nil
}

func NewCounterDevice(agentID string) modules.IHiveModule {
	m := &MyCounterDevice{
		Agent: *clientspkg.NewAgent(agentID, nil),
	}
	return m
}

func MyCounterModuleFactory(f factory.IModuleFactory) modules.IHiveModule {
	agentID := DefaultCounterDeviceThingID
	m := NewCounterDevice(agentID)
	return m
}

//

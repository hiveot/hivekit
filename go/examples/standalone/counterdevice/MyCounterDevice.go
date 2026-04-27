package counterdevice

import (
	_ "embed"
	"fmt"

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
const CounterDeviceThingID = "counterThingID"

// Affordance IDs
const (
	CounterPropName     = "counter"
	CounterUpdatedEvent = "counterUpdated"
	DecrementActionName = "decrement"
	IncrementActionName = "increment"
)

// The device is build on top of an agent.
// the agent is a thing itself.
// Agents facilitate storing and querying properties so you dont have to
type MyCounterDevice struct {
	clientspkg.Agent

	counter int
}

func (m *MyCounterDevice) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
	if req.ThingID != CounterDeviceThingID {
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
	m.counter--
	resp := req.CreateResponse(nil, nil)
	err = replyTo(resp)
	// m.PubEvent(req.ThingID, CounterUpdatedEvent, m.counter)
	return err
}
func (m *MyCounterDevice) HandleIncrement(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
	m.counter++
	resp := req.CreateResponse(nil, nil)
	err = replyTo(resp)
	// m.PubEvent(req.ThingID, CounterUpdatedEvent, m.counter)
	return err
}
func (m *MyCounterDevice) HandleWriteProperty(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
	var newValue int

	err = req.Decode(&newValue)
	if err == nil {
		// err = m.WriteProperty(req.ThingID, req.Name, newValue, true)
	}
	resp := req.CreateResponse(nil, err)
	err = replyTo(resp)
	if err == nil {
		// m.PubEvent(req.ThingID, CounterUpdatedEvent, m.counter)
	}
	return err
}

func MyCounterModuleFactory(f factory.IModuleFactory) modules.IHiveModule {
	m := &MyCounterDevice{}
	return m
}

//

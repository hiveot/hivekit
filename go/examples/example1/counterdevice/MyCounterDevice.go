package counterdevice

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/directory"
	"github.com/hiveot/hivekit/go/modules/factory"
	clientspkg "github.com/hiveot/hivekit/go/modules/transport/clients/pkg"
)

//go:embed "my-counter-tm.json"
var CounterDeviceTM []byte

// auto-increment the counter
const autoIncrementDelay = 10 * time.Second

// Module type for use in the recipe
const CounterDeviceModuleType = "counter-device"

// thingID requests are directed to
const DefaultCounterDeviceThingID = "counter1"

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

	counter          atomic.Int32
	backgroundCtx    context.Context
	backgroundCancel func()
}

func (m *MyCounterDevice) Background() {
	for {
		ctx, cancelFn := context.WithTimeout(m.backgroundCtx, autoIncrementDelay)
		<-ctx.Done()
		cancelFn()
		slog.Info("Incrementing counter")
		go m.Update(int(m.counter.Load() + 1))
	}
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
	resp := req.CreateResponse(nil, nil)
	if replyTo != nil {
		err = replyTo(resp)
	}
	go m.Update(int(m.counter.Load() - 1))
	return err
}

func (m *MyCounterDevice) HandleIncrement(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {

	resp := req.CreateResponse(nil, nil)
	if replyTo != nil {
		err = replyTo(resp)
	}
	go m.Update(int(m.counter.Load() + 1))
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
	if replyTo != nil {
		err = replyTo(resp)
	}
	if err == nil {
		// PubEvent makes the last event available via HandleReadRequests
		m.PubEvent(req.ThingID, CounterUpdatedEvent, m.counter.Load())
	}
	return err
}

// Start the device module.
func (m *MyCounterDevice) Start() error {
	m.backgroundCtx, m.backgroundCancel = context.WithCancel(context.Background())

	// publish the device TD
	// wait until the chain is complete before publishing the TD for discovery.
	go func() {
		time.Sleep(time.Millisecond)
		// write TD to the directory or discovery
		err := m.InvokeAction(directory.DefaultDirectoryThingID,
			directory.ActionCreateThing, CounterDeviceTM, nil)
		_ = err
	}()
	// publish the latest property values
	props := map[string]any{
		CounterPropName: m.counter.Load(),
	}
	thingID := m.GetModuleID()
	m.PubProperties(thingID, props)
	m.PubEvent(thingID, CounterUpdatedEvent, m.counter.Load())

	go m.Background()

	return nil
}

// stop the background process
func (m *MyCounterDevice) Stop() {
	slog.Info("Stopping counter")
	m.backgroundCancel()
}

// Update the counter and send a notification
func (m *MyCounterDevice) Update(newValue int) {
	m.counter.Store(int32(newValue))
	thingID := m.GetModuleID()
	// Send both a property update and event notification
	m.PubProperty(thingID, CounterPropName, m.counter.Load())
	m.PubEvent(thingID, CounterUpdatedEvent, m.counter.Load())
}

func NewCounterDevice(agentID string) modules.IHiveModule {
	m := &MyCounterDevice{
		Agent: *clientspkg.NewAgent(agentID, nil),
	}
	m.counter.Store(42)
	return m
}

func MyCounterModuleFactory(f factory.IModuleFactory) (modules.IHiveModule, error) {
	agentID := DefaultCounterDeviceThingID // must match the TD ID
	m := NewCounterDevice(agentID)
	return m, nil
}

//

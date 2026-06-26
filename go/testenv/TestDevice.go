package testenv

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
	"github.com/hiveot/hivekit/go/modules/agent"
	"github.com/hiveot/hivekit/go/modules/factory"
)

// TM of the test device
const counterDeviceTM = `
{
  "@context": [
    "https://www.w3.org/2022/wot/td/v1.1",
    {
      "hiveot": "https://www.hiveot.net/vocab/v0.1"
    }
  ],
  "@type": "Service",
  "base": "to be set by server",
  "id": "counter1",
  "title": "A simple counter",
  "description": "HiveKit test device that exposes a counter",
  "version": {
    "instance": "0.1.0"
  },
  "created": "2026-04-25T17:00:00.000Z",
  "modified": "2024-04-25T17:00:00.000Z",
  "support": "https://www.github.com/hiveot/hivekit",
  "properties": {
    "counter": {
      "title": "Current counter value",
      "type": "integer",
      "readonly": false
    }
  },
  "events": {
    "counterUpdated": {
      "title": "Counter changed",
      "description": "Event with the new counter value",
      "data": {
        "title": "New counter value",
        "type": "integer"
      }
    }
  },
  "actions": {
    "decrement": {
      "title": "Decrement the counter"
    },
    "increment": {
      "title": "Increment the counter"
    }
  }
}
`

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

type CounterConfig struct {
	// background counter
	AutoCount bool
	// reset the count if the auto-increment reaches this value
	ResetValue int
}

// Simple example of an IoT test device that tracks a counter.
// The device uses Agent as a base. Agents facilitate storing and querying properties so you dont have to.
//
// This implements the properties, events and actions listed in the device TM.
//
// To use this device it needs to be part of a chain:
// A. RC hub (no forms):  TestDevice -> transport client (wss,sse,mqtt)
// B. Standalone: http server -> transport server <-> authn service -> TestDevice -> discovery (TD)
type TestDevice struct {
	*agent.Agent

	config           *CounterConfig
	counter          atomic.Int32
	backgroundCtx    context.Context
	backgroundCancel func()
	tdocJson         string
}

// Run the counter in the background
func (m *TestDevice) Background() {
	for {
		if m.backgroundCtx.Err() != nil {
			return
		}
		ctx, cancelFn := context.WithTimeout(m.backgroundCtx, autoIncrementDelay)
		<-ctx.Done()
		cancelFn()
		slog.Info("Incrementing counter (in background)", "value", m.counter.Load())
		go m.Update(int(m.counter.Load() + 1))
	}
}

// Increment the counter
func (m *TestDevice) DoIncrement() {
	oldValue := m.counter.Load()
	if oldValue < int32(m.config.ResetValue) {
		m.counter.Store(oldValue + 1)
	} else {
		m.counter.Store(0)
	}
}

// Return the TD of this device.
// Forms should be added by the appropriate transport method used.
// This is also written to the directory on start.
func (m *TestDevice) GetTD() string {
	return m.tdocJson
}

func (m *TestDevice) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
	if req.ThingID != m.GetThingID() {
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

func (m *TestDevice) HandleDecrement(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
	resp := req.CreateResponse(nil, nil)
	if replyTo != nil {
		err = replyTo(resp)
	}
	go m.Update(int(m.counter.Load() - 1))
	return err
}

func (m *TestDevice) HandleIncrement(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {

	resp := req.CreateResponse(nil, nil)
	if replyTo != nil {
		err = replyTo(resp)
	}
	go m.DoIncrement()
	return err
}

func (m *TestDevice) HandleWriteProperty(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
	var newValue int

	err = req.Decode(&newValue)
	if err == nil {
		m.counter.Store(int32(newValue))
		// PubProperty makes the last value available via HandleReadRequests
		m.PubProperty(req.ThingID, req.Name, newValue, true)
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
func (m *TestDevice) Start() error {
	m.backgroundCtx, m.backgroundCancel = context.WithCancel(context.Background())

	tdoc, _ := td.UnmarshalTD(counterDeviceTM)
	tdoc.ID = m.GetThingID()
	m.tdocJson = td.MarshalTD(tdoc)

	// publish the device TD/TM
	// wait until the chain is complete before publishing the TD for discovery.
	// the transport will add the neccesary forms
	go func() {
		time.Sleep(time.Millisecond)
		// write TD to the directory or discovery
		// ignore the error if no directory/discovery exists in the chain
		err := m.WriteTD(m.tdocJson)
		_ = err
	}()
	// publish the latest property values
	props := map[string]any{
		CounterPropName: m.counter.Load(),
	}
	thingID := m.GetThingID()
	m.PubProperties(thingID, props, true)
	m.PubEvent(thingID, CounterUpdatedEvent, m.counter.Load())

	if m.config.AutoCount {
		go m.Background()
	}
	return nil
}

// stop the background process
func (m *TestDevice) Stop() {
	slog.Info("Stopping counter")
	m.backgroundCancel()
}

// Update the counter and send a notification
func (m *TestDevice) Update(newValue int) {
	m.counter.Store(int32(newValue))
	thingID := m.GetThingID()
	// Send both a property update and event notification
	m.PubProperty(thingID, CounterPropName, m.counter.Load(), true)
	m.PubEvent(thingID, CounterUpdatedEvent, m.counter.Load())
}

// Create a new counter device that starts counting at 42.
//
// the deviceID is the thingID
func NewTestDevice(deviceID string, config *CounterConfig) *TestDevice {
	if config == nil {
		config = &CounterConfig{}
	}
	m := &TestDevice{
		Agent:  agent.NewAgent(deviceID, nil),
		config: config,
	}
	m.counter.Store(42)
	return m
}

func NewTestDeviceFactory(f factory.IModuleFactory,
	md *factory.ModuleDefinition) (modules.IHiveModule, error) {

	// todo md can now provide configuration
	agentID := DefaultCounterDeviceThingID
	counterConfig, _ := md.Config.(*CounterConfig)
	m := NewTestDevice(agentID, counterConfig)
	return m, nil
}

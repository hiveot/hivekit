package testenv

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/cells/thing"
	"github.com/teris-io/shortid"
)

// TM of the test device
const counterThingTM = `
{
  "@context": [
    "https://www.w3.org/2022/wot/td/v1.1",
    {
      "hiveot": "https://www.hiveot.net/vocab/v0.1"
    }
  ],
  "@type": "Service",
  "base": "{{server}}",
  "id": "url:counter",
  "title": "A simple counter",
  "description": "HiveKit test Thing that exposes a counter",
  "version": {
    "instance": "0.1.0"
  },
  "created": "2026-06-25T17:00:00.000Z",
  "modified": "2026-06-25T17:00:00.000Z",
  "support": "https://www.github.com/hiveot/hivekit",
  "properties": {
    "autoincrement": {
      "title": "Auto Increment",
      "type": "bool"
    },
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

// Cell type for use in the recipe
const CounterThingCellType = "counter-thing"

// thingID requests are directed to
const DefaultTestCounterThingID = "counter1"

// Affordance IDs
const (
	AutoIncrementPropName = "autoincrement"
	CounterPropName       = "counter"
	CounterUpdatedEvent   = "counterUpdated"
	DecrementActionName   = "decrement"
	IncrementActionName   = "increment"
)

type CounterConfig struct {
	// background counter
	AutoIncrement bool
	// reset the count if the auto-increment reaches this value
	ResetValue int
}

// Simple example of an test counter 'Thing'.
// The device uses ExposedThing as a base as it facilitates storing and querying properties,
// so you dont have to.
//
// This Thing defines the properties, events and actions listed in the TM.
//
// To use this Thing it needs to be linked to a transport client (RC) or server:
// A. RC gateway (no forms needed):  CounterThing -> transport client (wss,sse,mqtt)
// B. Standalone: http server -> transport server <-> authn service -> CounterThing -> discovery (TD)
type TestCounterThing struct {
	*thing.ExposedThing

	config           *CounterConfig
	counter          atomic.Int32
	backgroundCtx    context.Context
	backgroundCancel func()
	tdocJson         string
}

// Run the counter in the background
func (m *TestCounterThing) Background() {
	for {
		if m.backgroundCtx.Err() != nil {
			return
		}
		ctx, cancelFn := context.WithTimeout(m.backgroundCtx, autoIncrementDelay)
		<-ctx.Done()
		cancelFn()
		slog.Info("Incrementing counter (in background)", "value", m.counter.Load())
		if m.config.AutoIncrement {
			go m.Update(int(m.counter.Load() + 1))
		}
	}
}

// Decrement the counter
func (m *TestCounterThing) DoDecrement() error {
	oldValue := m.counter.Load()
	newValue := oldValue - 1
	m.counter.Store(newValue)
	m.PubProperty("", CounterPropName, newValue, true)
	m.PubEvent("", CounterUpdatedEvent, newValue)
	return nil
}

// Increment the counter
func (m *TestCounterThing) DoIncrement() error {
	oldValue := m.counter.Load()
	newValue := oldValue + 1
	if oldValue >= int32(m.config.ResetValue) {
		newValue = 0
	}
	m.counter.Store(newValue)
	m.PubProperty("", CounterPropName, newValue, true)
	m.PubEvent("", CounterUpdatedEvent, newValue)
	return nil
}

// Return the TD of this device.
// Forms should be added by the appropriate transport method used.
// This is also written to the directory on start.
func (m *TestCounterThing) GetTD() string {
	return m.tdocJson
}

// Receive notifications from the chain
// * New connection to the server
// * Any notifications send by connected clients - none are expected so ignore these
func (m *TestCounterThing) HandleNotification(notif *msg.NotificationMessage) {
	if notif.AffordanceType == msg.AffordanceTypeEvent && notif.Name == api.ClientConnectionStatusEvent {
		slog.Info("HandleNotification: Client connection event", "data", notif.Data)
	}
	m.ForwardNotification(notif)
}

func (m *TestCounterThing) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
	if req.ThingID != m.GetThingID() {
		return m.ForwardRequest(req, replyTo)
	}
	// use Thing base to handle read properties/events/action requests
	err = m.HandleReadRequests(req, replyTo)
	if err == nil {
		return nil
	}

	// request was unhandled

	switch req.Operation {
	case td.OpInvokeAction:
		var output any

		switch req.Name {
		case DecrementActionName:
			err = m.DoDecrement()
		case IncrementActionName:
			err = m.DoIncrement()
		}
		resp := req.CreateResponse(output, err)
		err = replyTo(resp)
	case td.OpWriteProperty:
		return m.HandleWriteProperty(req, replyTo)
	default:
		err = fmt.Errorf("Unhandled operation '%s'", req.Operation)
	}
	return err
}

// Change a property value
func (m *TestCounterThing) HandleWriteProperty(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {

	switch req.Name {
	case CounterPropName:
		var newValue int
		err = req.DecodeInput(&newValue)
		if err == nil {
			m.counter.Store(int32(newValue))
			// PubProperty makes the last value available via HandleReadRequests
			m.PubProperty(req.ThingID, req.Name, newValue, true)
		}
	case AutoIncrementPropName:
		var newValue bool
		err = req.DecodeInput(&newValue)
		m.config.AutoIncrement = newValue
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

// Start the test device.
//
// This publishes a write TD request to the sink.
func (m *TestCounterThing) startDevice() error {
	m.backgroundCtx, m.backgroundCancel = context.WithCancel(context.Background())

	// Make the TD available. Set its thingID with the provided ID.
	tdoc, err := td.UnmarshalTD(counterThingTM)
	tdoc.ID = m.GetThingID()
	m.tdocJson = td.MarshalTD(tdoc)

	// publish the device TD
	// the downstream cells must already be started so writing the TD is
	// send to discovery or directory.
	go func() {
		time.Sleep(time.Millisecond)
		// write TD to the directory or discovery
		// ignore the error if no directory/discovery exists in the chain
		err := m.WriteTD(m.tdocJson)
		_ = err
	}()
	// publish the latest property values
	props := map[string]any{
		AutoIncrementPropName: m.config.AutoIncrement,
		CounterPropName:       m.counter.Load(),
	}
	thingID := m.GetThingID()
	m.PubProperties(thingID, props, true)
	m.PubEvent(thingID, CounterUpdatedEvent, m.counter.Load())

	if m.config.AutoIncrement {
		go m.Background()
	}
	return err
}

// stop the background process
func (m *TestCounterThing) Stop() {
	slog.Info("Stopping counter")
	m.backgroundCancel()
}

// Update the counter and send a notification
func (m *TestCounterThing) Update(newValue int) {
	m.counter.Store(int32(newValue))
	thingID := m.GetThingID()
	// Send both a property update and event notification
	m.PubProperty(thingID, CounterPropName, m.counter.Load(), true)
	m.PubEvent(thingID, CounterUpdatedEvent, m.counter.Load())
}

// Create a new counter exposed-thing that starts counting at 42.
//
// thingID is the thingID or use "" for an auto generated ID
// config defines behavior of the Thing
func StartTestCounterThing(thingID string, config *CounterConfig) (*TestCounterThing, error) {
	if config == nil {
		config = &CounterConfig{
			AutoIncrement: false,
			ResetValue:    1000,
		}
	}
	if thingID == "" {
		thingID = DefaultTestCounterThingID + "-" + shortid.MustGenerate()
	}
	m := &TestCounterThing{
		ExposedThing: thing.StartExposedThing(thingID, nil),
		config:       config,
	}
	m.counter.Store(42)

	err := m.startDevice()
	return m, err
}

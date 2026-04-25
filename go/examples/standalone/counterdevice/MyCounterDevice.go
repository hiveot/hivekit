package counterdevice

import (
	_ "embed"
	"fmt"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/clients"
	"github.com/hiveot/hivekit/go/modules/factory"
)

//go:embed "my-counter-tm.json"
var CounterDeviceTM []byte

// Module type for use in the recipe
const CounterDeviceModuleType = "counter-device"

// thingID requests are directed to
const CounterDeviceThingID = "counterThingID"

// The device is build on top of an agent.
// the agent is a thing itself.
// Agents facilitate storing and querying properties so you dont have to
type MyCounterDevice struct {
	clients.Agent

	counter int
}

func (m *MyCounterDevice) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
	if req.ThingID != CounterDeviceThingID {
		return m.ForwardRequest(req, replyTo)
	}
	err = m.HandleReadRequests(req, replyTo)
	if err == nil {
		return nil
	}
	// request was unhandled
	switch req.Operation {
	case td.OpInvokeAction:
		err = fmt.Errorf("TODO handle action '%s'", req.Name)
	case td.OpWriteProperty:
		err = fmt.Errorf("TODO handle write property '%s'", req.Name)
	default:
		err = fmt.Errorf("Unhandled operation '%s'", req.Operation)
	}
	return err
}

func MyCounterModuleFactory(f factory.IModuleFactory) modules.IHiveModule {
	m := &MyCounterDevice{}
	return m
}

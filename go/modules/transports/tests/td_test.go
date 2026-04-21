package transporttests

import (
	"errors"
	"testing"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/testenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// test TD messages and forms
// this uses the client and server helpers defined in connect_test.go

func TestAllTDProtocols(t *testing.T) {
	for _, testProtocol = range testProtocols {
		t.Run(testProtocol, TestAllTD)
	}
}
func TestAllTD(t *testing.T) {
	t.Run("TestAddForms", TestAddForms)
	// t.Run("TestPublishTD", TestPublishTD)
	t.Run("TestReadTDFromAgent", TestReadTDFromDevice)
}

const DeviceTypeSensor = "hiveot:sensor"

// Test consumer reads a TD from a device
// the device runs a server and offers a 'td' property
func TestReadTDFromDevice(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	var thingID = "thing1"
	var agentID = "agent1"
	var consumerID = "consumer1"

	// handler of TDs on the server
	// 1. start the transport
	testEnv, cancelFn := testenv.StartTestEnv(testProtocol)
	defer cancelFn()

	// 2. create an agent linked to this server
	ag1 := testEnv.NewServerAgent(agentID)
	defer ag1.Stop()

	// 3. agent creates TD
	td1 := td.NewTD(thingID, "My gadget", DeviceTypeSensor)
	td1.AddProperty("td", "Device TD", "", td.DataTypeString)

	// agent request handler to read the device TD from the td property.
	agentReqHandler := func(req *msg.RequestMessage, replyTo msg.ResponseHandler) error {
		t.Log("Received request: " + req.Operation)
		if req.Operation == td.OpReadProperty && req.Name == "td" {
			tdJSON := td1.ToString()
			resp := req.CreateResponse(tdJSON, nil)
			return replyTo(resp)
		} else {
			resp := req.CreateResponse(nil,
				errors.New("agent receives unknown request: "+req.Operation))
			return replyTo(resp)
		}
	}
	ag1.SetAppRequestHook(agentReqHandler)

	// 4. create a consumer and verify the TD can be read by a client
	co, cc, _ := testEnv.NewConsumerClient(consumerID, "somerole", nil)
	defer cc.Close()

	var rxTDJson string
	err := co.ReadPropertyAs(thingID, "td", &rxTDJson)
	require.NoError(t, err)

	rxTDoc, err := td.UnmarshalTD(rxTDJson)
	require.NoError(t, err)

	assert.Equal(t, td1.ID, rxTDoc.ID)
	assert.Equal(t, td1.Title, rxTDoc.Title)
}

// Test if forms are indeed added to a TD, describing the transport protocol binding operations
func TestAddForms(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	var thingID = "thing1"

	// handler of TDs on the server
	// 1. start the transport
	testEnv, cancelFn := testenv.StartTestEnv(testProtocol)
	defer cancelFn()

	// 2. Create a TD
	// tdi := td.NewTD(thingID, "My gadget", DeviceTypeSensor)
	tdi := testEnv.CreateTestTD(1)
	tdi.ID = thingID

	// 3. add forms
	testEnv.Server.AddTDSecForms(tdi, true)

	// 4. Check that at least 1 form are present
	assert.GreaterOrEqual(t, len(tdi.Forms), 1)

	// 5. Expect security scheme
	require.NotEmpty(t, tdi.Security)
	require.NotEmpty(t, tdi.SecurityDefinitions)

	scheme, err := tdi.GetSecurityScheme()
	assert.NoError(t, err)
	assert.NotEmpty(t, scheme.Scheme)

}

//// Agent Publishes TD to the directory
//func TestPublishTD(t *testing.T) {
//	t.Logf("---%s---\n", t.Name())
//	var thingID = "thing1"
//	var rxTD atomic.Value
//
//	// 2. Create a TD
//	td1 := td.NewTD(thingID, "My gadget", DeviceTypeSensor)
//	td1JSON, _ := jsoniter.MarshalToString(td1)
//
//	// handler of TDs on the server
//	requestHandler := func(msg *transports.RequestMessage,
//		c transports.IConnection) *transports.ResponseMessage {
//		var err error
//		if msg.Operation == td.HTOpUpdateTD {
//			assert.Equal(t, thingID, msg.ThingID)
//			assert.Equal(t, td1JSON, msg.Input)
//			assert.NotEmpty(t, msg.Input)
//			rxTD.Store(msg.Input)
//		} else {
//			err = fmt.Errorf("Unexpected operation: %s" + msg.Operation)
//		}
//		resp := msg.CreateResponse(nil, err)
//		return resp
//	}
//
//	// 1. start the transport server with the TD handler
//	srv, cancelFn := StartTransportServer(requestHandler, nil)
//	_ = srv
//	defer cancelFn()
//
//	// 2. Connect as agent
//	cc1, ag1, _ := NewAgent(testAgentID1)
//	defer cc1.Disconnect()
//
//	// Agent publishes the TD
//	err := ag1.UpdateThing(td1)
//	require.NoError(t, err)
//	time.Sleep(time.Millisecond * 10)
//
//	// check reception
//	require.Equal(t, td1JSON, rxTD.Load())
//}

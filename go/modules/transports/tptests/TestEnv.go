package tptests

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path"
	"time"

	"github.com/hiveot/hivekit/go/modules/certs/module/selfsigned"
	"github.com/hiveot/hivekit/go/modules/clients"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/modules/transports/httpbasic/httpbasicserver"
	"github.com/hiveot/hivekit/go/modules/transports/httpserver"
	"github.com/hiveot/hivekit/go/modules/transports/httpserver/module"
	httpmodule "github.com/hiveot/hivekit/go/modules/transports/httpserver/module"
	ssescserver "github.com/hiveot/hivekit/go/modules/transports/ssesc/server"
	"github.com/hiveot/hivekit/go/modules/transports/wss/wssserver"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hivekit/go/vocab"
	"github.com/hiveot/hivekit/go/wot/td"
)

const (
	TestServerHttpPort = 9445
	TestTimeout        = time.Minute
)

// testTDs are a bunch of TD's for generating test data. The first 5 are predefined and always the same.
// A higher number generates at random.
// See CreateTestTD for details.
var testTDs = []struct {
	ID         string
	Title      string
	DeviceType string
	NrEvents   int
	NrProps    int
	NrActions  int
}{
	{ID: "thing-1", Title: "Environmental Sensor",
		DeviceType: vocab.ThingSensorEnvironment, NrEvents: 1, NrProps: 1, NrActions: 3},
	{ID: "thing-2", Title: "Light Switch",
		DeviceType: vocab.ThingActuatorLight, NrEvents: 2, NrProps: 2, NrActions: 0},
	{ID: "thing-3", Title: "Power meter",
		DeviceType: vocab.ThingMeterElectric, NrEvents: 3, NrProps: 3, NrActions: 1},
	{ID: "thing-4", Title: "Multisensor",
		DeviceType: vocab.ThingSensorMulti, NrEvents: 4, NrProps: 4, NrActions: 2},
	{ID: "thing-5", Title: "Alarm",
		DeviceType: vocab.ThingActuatorAlarm, NrEvents: 2, NrProps: 2, NrActions: 2},
}

var PropTypes = []string{vocab.PropDeviceMake, vocab.PropDeviceModel,
	vocab.PropDeviceDescription, vocab.PropDeviceFirmwareVersion, vocab.PropLocationCity}
var EventTypes = []string{vocab.PropElectricCurrent, vocab.PropElectricVoltage,
	vocab.PropElectricPower, vocab.PropEnvTemperature, vocab.PropEnvPressure}
var ActionTypes = []string{vocab.ActionDimmer, vocab.ActionSwitch,
	vocab.ActionSwitchToggle, vocab.ActionValveOpen, vocab.ActionValveClose}

// Test environment for testing modules
type TestEnv struct {
	// Authenticator to use for managing clients
	Authenticator transports.IAuthenticator
	// certificate bundle to use for this test environment
	CertBundle selfsigned.TestCertBundle
	// base http server
	HttpServer *httpmodule.HttpServerModule
	// The transport server connection URL
	ServerURL string
	// the transport to use for this test environment
	Server transports.ITransportServer
	// the storage root directory to use by modules. Module add their moduleID to the path.
	StorageRoot string
}

// CreateTestTD returns a test TD with ID "thing-{i}", and a variable
// number of properties, events and actions.
//
//	properties are named "prop-{i}
//	events are named "event-{i}
//	actions are named "action-{i}
//
// The first 10 are predefined and always the same. A higher number generates at random.
// i is the index.
func (testEnv *TestEnv) CreateTestTD(i int) (tdi *td.TD) {
	ttd := testTDs[0]
	if i < len(testTDs) {
		ttd = testTDs[i]
	} else {
		ttd.ID = fmt.Sprintf("thing-%d", rand.Intn(99823))
	}

	tdi = td.NewTD("", ttd.ID, ttd.Title, ttd.DeviceType)
	// add random properties
	for n := 0; n < ttd.NrProps; n++ {
		propName := fmt.Sprintf("prop-%d", n)
		tdi.AddProperty(propName, "title-"+PropTypes[n], "", vocab.WoTDataTypeString).
			SetAtType(PropTypes[n])
	}
	// add random events
	for n := 0; n < ttd.NrEvents; n++ {
		evName := fmt.Sprintf("event-%d", n)
		tdi.AddEvent(evName, "title-"+EventTypes[n], "",
			&td.DataSchema{Type: vocab.WoTDataTypeString}).
			SetAtType(EventTypes[n])
	}
	// add random actions
	for n := 0; n < ttd.NrActions; n++ {
		actionName := fmt.Sprintf("action-%d", n)
		tdi.AddAction(actionName, "title-"+ActionTypes[n], "",
			&td.DataSchema{Type: vocab.WoTDataTypeString},
		).SetAtType(ActionTypes[n])
	}

	return tdi
}

// NewClient creates a new connected client with the given client ID.
//
// This creates an account and access token for the client if needed.
//
// This panics if a client cannot be created or cannot connect.
func (testEnv *TestEnv) NewClient(clientID string, role string, ch transports.ConnectionHandler) (
	clients.IClientModule, string) {

	// ensure the test client account exists
	err := testEnv.Authenticator.AddClient(clientID, clientID, role, "")
	token, _, err := testEnv.Authenticator.CreateToken(clientID, time.Minute*10)
	if err != nil {
		panic("NewClient: createToken failed: " + err.Error())
	}
	// create a connection to the test server
	cl, err := clients.NewTransportClient(
		testEnv.ServerURL, testEnv.CertBundle.CaCert)
	if err == nil {
		cl.SetTimeout(TestTimeout)
		err = cl.ConnectWithToken(clientID, token, ch)
	}
	if err != nil {
		panic("NewClient failed to connect:" + err.Error())
	}
	return cl, token
}

// NewServerAgent creates a new agent that is a direct sink for the test server.
// Additional agents can be chained by setting them as the sink of the previous agent.
//
// An account for the agent is created and the agent is set as the request sink for the
// server.
//
// This panics if the agent cannot be created.
func (testEnv *TestEnv) NewServerAgent(agentID string) *clients.Agent {

	// Simple server side agent. No account needed
	agent := clients.NewAgent(agentID, nil)

	// the agent is the sink for the server
	testEnv.Server.SetRequestSink(agent.HandleRequest)
	agent.SetNotificationSink(testEnv.Server.SendNotification)
	return agent
}

// NewRCAgent creates a new reverse-connection agent/consumer with the given ID.
// This uses connection reversal where the agent connects as a client to the server.
//
// The agent is the sink of the client. Requests received via the client are handled
// by the agent. Notifications generated by the agent are send to the client for transmission
// to the server.
//
// Note that if the agent is also a consumer, its sink has to be set to the same client
// that forwards its notifications.
//
// This returns the agent module, its client connection and the auth token.
// This panics if a client cannot be created
func (testEnv *TestEnv) NewRCAgent(clientID string) (
	ag *clients.Agent, cc transports.IConnection, authToken string) {

	// cc is the client connection for the agent that receives requests from the
	// server for the agent and sends notifications to the server.
	cl, authToken := testEnv.NewClient(clientID, transports.ClientRoleAgent, nil)

	// simple agent, no application request handler yet
	agent := clients.NewAgent(clientID+"-agent", nil)

	// the client delivers requests to the agent and receives notifications from it
	cl.SetRequestSink(agent.HandleRequest)
	agent.SetNotificationSink(cl.SendNotification)

	// When acting in a dual role as agent and consumer, the agent uses the client as
	// the sink for requests and receives notifications passed to the client from the server.
	agent.SetRequestSink(cl.HandleRequest)
	cl.SetNotificationSink(agent.HandleNotification)

	return agent, cl, authToken
}

// NewConsumerClient creates a new connected consumer.
// The transport server must be started first.
//
// This uses the clientID as password
// This panics if a client cannot be created
//
//	clientID to use
//	role of the client
//	optional connection change callback
func (testEnv *TestEnv) NewConsumerClient(
	clientID string, role string, ch transports.ConnectionHandler) (
	co *clients.Consumer, cc clients.IClientModule, token string) {

	cc, token = testEnv.NewClient(clientID, role, ch)

	co = clients.NewConsumer(clientID + "-consumer")
	co.SetRequestSink(cc.HandleRequest)
	co.SetTimeout(TestTimeout)
	// notifications received by the client are passed to the consumer
	cc.SetNotificationSink(co.HandleNotification)
	return co, cc, token
}

// Create a new running test server instance using the given http server
//
// protocols is one of a list of the server protocols to support. nil for all
// protocols:
// * transports.ProtocolTypeHTTPBasic
// * ProtocolTypeWotWSS
// * ProtocolTypeHiveotSSE
// * ProtocolTypePassthrough
func (testEnv *TestEnv) StartTestServer(protocol string) (srv transports.ITransportServer) {

	var err error

	switch protocol {
	case transports.ProtocolTypeHTTPBasic:

		srv = httpbasicserver.NewHttpBasicServer(testEnv.HttpServer)
		err = srv.Start("")
		// http only, no subprotocol bindings

	case transports.ProtocolTypeHiveotSSE:
		srv = ssescserver.NewHiveotSsescServer(testEnv.HttpServer, nil)
		err = srv.Start("")

	case transports.ProtocolTypeWotWSS:
		srv = wssserver.NewWotWssServer(testEnv.HttpServer)
		err = srv.Start("")

	default:
		err = errors.New("unknown protocol name: " + protocol)
	}

	if err != nil {
		panic("Unable to create transport server module: " + err.Error())
	}
	testEnv.ServerURL = srv.GetConnectURL()

	return srv
}

// Start a http server module with default port, test certs and dummy authenticator
// This server is needed for http-basic, websocket, hiveot-sse-sc subprotocols
// Also used to serve http endpoints for the directory and authn users.
// This panic if the server cannot be started.
func (testEnv *TestEnv) StartHttpServer() {

	// cert uses localhost
	cfg := httpserver.NewHttpServerConfig(
		testEnv.CertBundle.ServerAddr, TestServerHttpPort,
		testEnv.CertBundle.ServerCert,
		testEnv.CertBundle.CaCert,
		testEnv.Authenticator.ValidateToken)

	// cfg.Address = fmt.Sprintf("%s:%d", certBundle.ServerAddr, testServerHttpPort)

	testEnv.HttpServer = module.NewHttpServerModule("", cfg)
	err := testEnv.HttpServer.Start()
	if err != nil {
		panic("unable to start TLS server: " + err.Error())
	}
	testEnv.ServerURL = testEnv.HttpServer.GetConnectURL()
}

// NewTestEnv creates a new test environment with certificates and a dummy authenticator.
// This does not start any servers.
// Use StartHttpServer,StartTestServer to initialize startup the servers.
// This creates a HTTP server, protocol server, certificates and a dummy authenticator
// This sets the storage root directory to {os.TempDir}/hivekit
func NewTestEnv() *TestEnv {
	testEnv := &TestEnv{
		CertBundle:    selfsigned.CreateTestCertBundle(utils.KeyTypeED25519),
		Authenticator: NewDummyAuthenticator(),
		StorageRoot:   path.Join(os.TempDir(), "hivekit"),
	}
	return testEnv
}

// StartTestEnv start a new test environment for the given transport protocol.
// This creates a HTTP server, protocol server, certificates and a dummy authenticator
func StartTestEnv(protocol string) (testEnv *TestEnv, cancelFunc func()) {
	testEnv = NewTestEnv()
	testEnv.StartHttpServer()
	testEnv.Server = testEnv.StartTestServer(protocol)
	return testEnv, func() {
		testEnv.Server.Stop()
		testEnv.HttpServer.Stop()
	}
}

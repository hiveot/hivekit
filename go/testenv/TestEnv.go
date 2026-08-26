package testenv

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/api/vocab"
	"github.com/hiveot/hivekit/go/cells/authn"
	certstest "github.com/hiveot/hivekit/go/cells/certs/test"
	"github.com/hiveot/hivekit/go/cells/consumer"
	reconnect_service "github.com/hiveot/hivekit/go/cells/reconnect/service"
	"github.com/hiveot/hivekit/go/cells/thing"
	"github.com/hiveot/hivekit/go/cells/transport/clients"
	grpc_server "github.com/hiveot/hivekit/go/cells/transport/grpc/server"
	httpbasic_server "github.com/hiveot/hivekit/go/cells/transport/httpbasic/server"
	ssesc_server "github.com/hiveot/hivekit/go/cells/transport/ssesc/server"
	"github.com/hiveot/hivekit/go/cells/transport/tlsserver"
	tls_server "github.com/hiveot/hivekit/go/cells/transport/tlsserver/server"
	wss_server "github.com/hiveot/hivekit/go/cells/transport/wss/server"
	"github.com/hiveot/hivekit/go/utils"
)

const (
	TestServerHttpPort = 9445
	TestGrpcTcpPort    = 50052 // where 50051 is the common port
	TestTimeout        = time.Minute * 3
)

var TestHome = filepath.Join(os.TempDir(), "hivekit-test")
var TestUDSPath = "/tmp/hivekit/testenv.socket"
var TestGrpcUnixURL = api.HiveotGrpcUnixScheme + "://" + TestUDSPath
var TestGrpcTcpURL = fmt.Sprintf("%s://localhost:%d", api.HiveotGrpcTcpScheme, TestGrpcTcpPort)

// alt UDS using TCP socket
// var TestUDSPath = ":8899"
// var TestUDSURL = "tcp://" + TestUDSPath

// var DefaultProtocol = api.HiveotWebsocketProtocolType

// var DefaultProtocol = api.HiveotGrpcTcpProtocolType

var DefaultProtocol = api.HiveotSseScProtocolType

// var DefaultProtocol = api.WotWebsocketProtocolType

// var DefaultProtocol = api.HttpBasicProtocolType

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
		DeviceType: vocab.DeviceSensorEnvironment, NrEvents: 1, NrProps: 1, NrActions: 3},
	{ID: "thing-2", Title: "Light Switch",
		DeviceType: vocab.DeviceActuatorLight, NrEvents: 2, NrProps: 2, NrActions: 0},
	{ID: "thing-3", Title: "Power meter",
		DeviceType: vocab.DeviceMeterElectric, NrEvents: 3, NrProps: 3, NrActions: 1},
	{ID: "thing-4", Title: "Multisensor",
		DeviceType: vocab.DeviceSensorMulti, NrEvents: 4, NrProps: 4, NrActions: 2},
	{ID: "thing-5", Title: "Alarm",
		DeviceType: vocab.DeviceActuatorAlarm, NrEvents: 2, NrProps: 2, NrActions: 2},
}

var PropTypes = []string{vocab.PropDeviceMake, vocab.PropDeviceModel,
	vocab.PropDeviceDescription, vocab.PropDeviceFirmwareVersion, vocab.PropLocationCity}
var EventTypes = []string{vocab.PropElectricCurrent, vocab.PropElectricVoltage,
	vocab.PropElectricPower, vocab.PropEnvTemperature, vocab.PropEnvPressure}
var ActionTypes = []string{vocab.ActionDimmer, vocab.ActionSwitch,
	vocab.ActionSwitchToggle, vocab.ActionValveOpen, vocab.ActionValveClose}

// Test environment for testing cells
type TestEnv struct {
	// App test environment with directories
	AppEnv *api.HiveEnvironment
	// certificate bundle to use for this test environment
	CertBundle certstest.TestCertBundle
	// base http server
	HttpServer api.IHttpServer
	// The transport server connection URL
	ServerURL string
	// the transport to use for this test environment
	Server         api.ITransportServer
	ServerProtocol string
	// Authenticator to use for managing clients
	TestAuthn *TestAuthenticator
}

// CreateTestTD returns a test TD with ID "thing-{i}", and a variable
// number of properties, events and actions.
//
//	properties are named "prop-{j}
//	events are named "event-{j}
//	actions are named "action-{j}
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

	tdi = td.NewTD(ttd.ID, ttd.Title, ttd.DeviceType)
	// add random properties
	for n := 0; n < ttd.NrProps; n++ {
		propName := fmt.Sprintf("prop-%d", n)
		tdi.AddProperty(propName, "title-"+PropTypes[n], "", td.DataTypeString).
			SetAtType(PropTypes[n])
	}
	// add random events
	for n := 0; n < ttd.NrEvents; n++ {
		evName := fmt.Sprintf("event-%d", n)
		tdi.AddEvent(evName, "title-"+EventTypes[n], "",
			&td.DataSchema{Type: td.DataTypeString}).
			SetAtType(EventTypes[n])
	}
	// add random actions
	for n := 0; n < ttd.NrActions; n++ {
		actionName := fmt.Sprintf("action-%d", n)
		tdi.AddAction(actionName, "title-"+ActionTypes[n], "",
			&td.DataSchema{Type: td.DataTypeString},
		).SetAtType(ActionTypes[n])
	}

	return tdi
}

// create a new authentication token
func (testEnv *TestEnv) CreateToken(clientID string, validity time.Duration) (token string, validUntil time.Time, err error) {
	token, validUntil, err = testEnv.TestAuthn.CreateToken(clientID, validity)
	return token, validUntil, err
}

// NewConnectedClient creates a new reverse-connected client with the given client ID.
//
// This creates an account and access token for the client if needed.
//
// This panics if a client cannot be created or cannot connect.
func (testEnv *TestEnv) NewConnectedClient(
	clientID string, role string) (cl api.ITransportClient, token string) {

	// ensure the test client account exists
	err := testEnv.TestAuthn.AddClient(clientID, clientID, role)
	token, _, err = testEnv.CreateToken(clientID, time.Minute*10)
	if err != nil {
		panic("NewConnectedClient: createToken failed: " + err.Error())
	}
	// create a connection to the test server
	serverTD := testEnv.Server.GetTD()
	form, _ := serverTD.GetConnectForm("", "")
	cl, err = clients.NewTransportClientFromForm(serverTD, form, testEnv.CertBundle.RootCAs)

	// disable forwarding. This should not make a difference unless Forward... is used instead of Emit...
	// cl.SetForwarding(false, false)

	if err == nil {
		cl.SetTimeout(TestTimeout)
		err = cl.SetAuthToken(clientID, token, td.SecSchemeBearer)
	}
	if err == nil {
		err = cl.Connect()
	}
	if err != nil {
		panic("NewConnectedClient failed to connect:" + err.Error())
	}
	return cl, token
}

// NewServerThing creates an exposed thing that is a direct sink for the test server.
// Additional cells can be chained by setting them as the sink of the previous cells.
//
// An account for the thing is created and the thing is set as the request sink for the
// server.
//
// This panics if the thing cannot be created.
func (testEnv *TestEnv) NewServerThing(thingID string) *thing.ExposedThing {

	// Simple server side Thing. No account needed
	m := thing.NewExposedThing(thingID, nil)

	// the device is the request sink for the transport server
	testEnv.Server.SetRequestSink(m)
	m.SetNotificationSink(testEnv.Server)
	return m
}

// NewRCThing creates a new reverse-connection thing with the given ID.
// This uses connection reversal where the thing connects as a client to the server.
//
// The Thing is set as the client connection request sink. Requests received via the
// client are passed to the thing.
// The client connection is set as the thing notification sink so notifications sent
// to the server.
//
// To allow Things to act as a consumer, its request sink is set to the client connection
// and the Thing is set as the notification sink for the connection.
// Note that the Thing should have an appRequest handler set to avoid request looping.
//
// This returns the exposed Thing, its connected client connection and the auth token.
// This panics if a client cannot be created
func (testEnv *TestEnv) NewRCThing(clientID string, appReqHandler msg.RequestHandler) (
	ag *thing.ExposedThing, cc api.IConnection, authToken string) {

	// cc is the client connection for the Thing that receives requests from the
	// server and sends notifications to the server.
	cl, authToken := testEnv.NewConnectedClient(clientID, authn.ClientRoleDevice)

	// simple ething, no application request handler yet
	ething := thing.NewExposedThing(clientID+"-thing", appReqHandler)

	// the client delivers requests to the thing and receives notifications from it
	cl.SetRequestSink(ething)
	ething.SetNotificationSink(cl)

	// When acting in a dual role as thing and consumer, the thing uses the client as the
	// sink for requests and receives notifications passed to the client from the server.
	// forwarding must be disabled to prevent received notifications to be sent back to the
	// client.
	ething.SetRequestSink(cl)
	cl.SetNotificationSink(ething)
	// disable forwarding to prevent received notifications to be forwarded back
	// to the client.
	ething.SetForwarding(false, false)

	return ething, cl, authToken
}

// NewConnectedConsumer creates a new connected consumer.
// The transport server must be started first so the client can connect.
//
// This uses the clientID as password
// This panics if a client cannot be created
//
//	clientID to use
//	role of the client
func (testEnv *TestEnv) NewConnectedConsumer(clientID string, role string) (
	co *consumer.Consumer, cc api.ITransportClient, token string) {

	cc, token = testEnv.NewConnectedClient(clientID, role)
	co = consumer.NewConsumer(cc, nil)
	co.SetTimeout(TestTimeout)
	return co, cc, token
}

// NewReconnectedConsumer creates a new connected consumer with the reconnect capability.
// The reconnect service is placed before client.
// The transport server must be started first so that connect can succeed.
//
// This uses the clientID as password
// This panics if a client cannot be created
//
//	clientID to use
//	role of the client
func (testEnv *TestEnv) NewReconnectedConsumer(clientID string, role string) (
	co *consumer.Consumer, cc api.ITransportClient, token string) {

	cc, token = testEnv.NewConnectedClient(clientID, role)

	// insert the reconnect service between consumer and client connection
	rc := reconnect_service.NewReconnectService(cc)

	co = consumer.NewConsumer(rc, nil)
	co.SetTimeout(TestTimeout)

	return co, cc, token
}

// Create a new running test transport server .
//
// This can be called multiple times to support multiple servers. However, only the
// first server will be stored in the 'TestEnv.Server' property.
//
// protocols is one of a list of the server protocols to support. "" for default.
// protocols:
// * api.ProtocolTypeWotHTTPBasic
// * ProtocolTypeWotWSS
// * ProtocolTypeHiveotSSE
// * and more
func (testEnv *TestEnv) StartTestServer(protocol string) (srv api.ITransportServer) {

	var err error
	if protocol == "" {
		protocol = DefaultProtocol
	}

	switch protocol {
	case api.HiveotGrpcTcpProtocolType:
		serverCert := testEnv.CertBundle.ServerCert
		caCert := testEnv.CertBundle.CaCert
		srv = grpc_server.NewHiveotGrpcServer(
			TestGrpcTcpURL, serverCert, caCert, testEnv.TestAuthn, TestTimeout)
		err = srv.Start()
	case api.HiveotGrpcUnixProtocolType:
		serverCert := testEnv.CertBundle.ServerCert
		caCert := testEnv.CertBundle.CaCert
		srv = grpc_server.NewHiveotGrpcServer(
			TestGrpcUnixURL, serverCert, caCert, testEnv.TestAuthn, TestTimeout)
		err = srv.Start()

	case api.HiveotSseScProtocolType:
		testEnv.StartHttpServer(false)
		srv = ssesc_server.NewSseScServer(testEnv.HttpServer, TestTimeout)
		err = srv.Start()

	case api.HiveotWebsocketProtocolType:
		testEnv.StartHttpServer(false)
		srv = wss_server.NewHiveotWssServer(testEnv.HttpServer, TestTimeout)
		err = srv.Start()

	case api.HttpBasicProtocolType:
		testEnv.StartHttpServer(false)
		srv = httpbasic_server.NewHttpBasicServer(testEnv.HttpServer)
		err = srv.Start()
		// http only, no subprotocol bindings

	case api.WotWebsocketProtocolType:
		testEnv.StartHttpServer(false)
		srv = wss_server.NewWotWssServer(testEnv.HttpServer, TestTimeout)
		err = srv.Start()

	default:
		err = errors.New("StartTestServer: unknown protocol name: " + protocol)
	}
	// avoid unnecesary notification warnings as notifications created by the server can be ignored.
	// srv.SetNotificationSink(func(*msg.NotificationMessage) { /*dummy*/ })

	if err != nil {
		panic("StartTestServer: Unable to create transport server service: " + err.Error())
	}
	// dont override the first transport server in case multiple transports are used
	if testEnv.Server == nil {
		testEnv.Server = srv
		testEnv.ServerProtocol = protocol
		testEnv.ServerURL = srv.GetConnectURL()
	}
	return srv
}

// Start a http transport server with default port, test certs and dummy authenticator
//
// This server is needed for http-basic, websocket, hiveot-sse-sc subprotocols
// Also used to serve http endpoints for the directory and authn users.
//
// If the http server is already running then do nothing.
// This returns the http server and its URL
// This panic if the server cannot be started.
func (testEnv *TestEnv) StartHttpServer(logging bool) (api.IHttpServer, string) {
	if testEnv.HttpServer != nil {
		return testEnv.HttpServer, testEnv.ServerURL
	}
	// cert uses localhost
	cfg := tlsserver.NewTLSServerConfig(
		testEnv.CertBundle.ServerAddr, testEnv.AppEnv.HttpsPort,
		testEnv.CertBundle.ServerCert,
		testEnv.CertBundle.RootCAs,
		logging)

	// cfg.Address = fmt.Sprintf("%s:%d", certBundle.ServerAddr, testServerHttpPort)

	testEnv.HttpServer = tls_server.NewTLSServer(cfg, testEnv.TestAuthn)
	err := testEnv.HttpServer.Start()
	if err != nil {
		panic("unable to start TLS server: " + err.Error())
	}
	testEnv.ServerURL = testEnv.HttpServer.GetConnectURL()
	return testEnv.HttpServer, testEnv.ServerURL
}

// stop running servers
func (testEnv *TestEnv) Stop() {
	if testEnv.Server != nil {
		testEnv.Server.Stop()
		testEnv.Server = nil
	}
	if testEnv.HttpServer != nil {
		testEnv.HttpServer.Stop()
		testEnv.HttpServer = nil
	}
}

// NewTestEnv creates a new test environment with certificates and a dummy authenticator.
//
// This does not start any servers.
// Use StartHttpServer,StartTestServer to initialize startup the servers.
// This creates a HTTP server, protocol server, certificates and a dummy authenticator
// This sets the storage root directory to {os.TempDir}/hivekit
//
// if clean is set then delete the content of the home folder for a clean start.
func NewTestEnv(clean bool) *TestEnv {
	if clean {
		os.RemoveAll(TestHome)
		os.MkdirAll(TestHome, 0750)
	}
	appEnv := api.NewHiveEnvironment(TestHome, false)
	utils.SetLogging("info", "")
	appEnv.HttpsPort = TestServerHttpPort
	// ensure the directories exist
	os.MkdirAll(appEnv.BinDir, 0750)
	os.MkdirAll(appEnv.CertsDir, 0700)
	os.MkdirAll(appEnv.ConfigDir, 0750)
	os.MkdirAll(appEnv.LogsDir, 0750)
	// os.MkdirAll(appEnv.PluginsDir, 0750)
	os.MkdirAll(appEnv.StoresDir, 0750)
	certBundle := certstest.CreateTestCertBundle(utils.KeyTypeED25519)
	appEnv.SetCACert(certBundle.CaCert)
	testEnv := &TestEnv{
		AppEnv:     appEnv,
		CertBundle: certBundle,
		TestAuthn:  NewTestAuthenticator(),
	}
	return testEnv
}

// StartTestEnv start a new test environment for the given transport protocol type.
// If no protocol type is provided this uses the default protocol (top of this file.)
// This starts a HTTP server, protocol server, certificates and a test authenticator
// if clean is set then delete the content of the home folder for a clean start.
func StartTestEnv(protocol string, clean bool) (testEnv *TestEnv, cancelFunc func()) {
	testEnv = NewTestEnv(clean)
	testEnv.StartHttpServer(true)
	testEnv.Server = testEnv.StartTestServer(protocol)
	return testEnv, func() {
		// give connections time to close client side before forcing them to close server side
		time.Sleep(time.Millisecond)
		testEnv.Server.Stop()
		testEnv.HttpServer.Stop()
	}
}

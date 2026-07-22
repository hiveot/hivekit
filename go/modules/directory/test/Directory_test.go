package directory_test

import (
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules/authn"
	"github.com/hiveot/hivekit/go/modules/directory"
	directory_client "github.com/hiveot/hivekit/go/modules/directory/client"
	directory_service "github.com/hiveot/hivekit/go/modules/directory/service"
	tls_client "github.com/hiveot/hivekit/go/modules/transport/tlsclient/client"
	"github.com/hiveot/hivekit/go/testenv"
	"github.com/hiveot/hivekit/go/utils"
	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var storageDir = filepath.Join(os.TempDir(), "hivekit", "directory-test")

const defaultDeviceID = "device-smith"
const defaultProtocol = api.ProtocolTypeWotWebsocket
const TestKeyType = utils.KeyTypeED25519
const rpcTimeout = time.Minute // for testing/debugging

// TestMain setsup logging
func TestMain(m *testing.M) {
	utils.SetLogging("info", "")

	result := m.Run()
	if result != 0 {
		println("Test failed with code:", result)
	} else {
	}

	os.Exit(result)
}

// Start a test environment with a directory module connected to the server.
// withHttp means that Directory HTTP API is started for serving directory requests over http.
func StartDirectoryServer(withHttp bool) (
	testEnv *testenv.TestEnv, m directory.IDirectoryService, cancelFn func()) {

	var dirHttpServer directory.IDirectoryHttpServer

	proto := defaultProtocol
	testEnv, cancelTestEnv := testenv.StartTestEnv(proto, true)
	transports := []api.ITransportServer{testEnv.Server}

	if withHttp {
		// add directory endpoints to the http server
		dirHttpServer = directory_service.NewDirectoryHttpServer(testEnv.HttpServer, rpcTimeout)
		transports = append(transports, dirHttpServer)
	}
	// the transports are used to update the TDD forms and security
	m = directory_service.NewDirectoryService("", storageDir, testEnv.HttpServer, transports)
	err := m.Start()
	if err != nil {
		panic("StartDirectoryServer: failed to start the directory " + err.Error())
	}
	if withHttp {
		dirHttpServer.SetRequestSink(m)
	}

	// http requests are passed as RRN messages to the directory server
	// if httpAPI != nil {
	// httpAPI.SetRequestSink(m.HandleRequest)
	// }
	// RRN requests from the server are passed as RRN to the directory server
	testEnv.Server.SetRequestSink(m)
	// the server receives the notification and sends them to remote clients
	m.SetNotificationSink(testEnv.Server)
	return testEnv, m, func() {
		if dirHttpServer != nil {
			dirHttpServer.Stop()
		}
		m.Stop()
		cancelTestEnv()
	}
}

// Generic directory store testcases
func TestStartStop(t *testing.T) {
	t.Logf("---%s---\n", t.Name())

	m := directory_service.NewDirectoryService("", storageDir, nil, nil)
	err := m.Start()
	require.NoError(t, err)
	defer m.Stop()

	// add a thing
	tdJson := directory.DirectoryTMJson
	m.UpdateThing(defaultDeviceID, string(tdJson))

	// read all things
	tdList, err := m.RetrieveAllThings(0, 10)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(tdList), 1)
}

func TestCreateTD(t *testing.T) {
	thingID := "thing1"

	m := directory_service.NewDirectoryService("", storageDir, nil, nil)
	err := m.Start()
	require.NoError(t, err)
	defer m.Stop()

	// add the directory itself
	tdJson := directory.DirectoryTMJson
	m.UpdateThing(defaultDeviceID, string(tdJson))

	// read all things, expect 1
	tdList, err := m.RetrieveAllThings(0, 10)
	assert.NoError(t, err)
	assert.Len(t, tdList, 1)

	// add another TD
	tdi1 := td.NewTD(thingID, "test thing", "test device")
	td1Json := tdi1.ToString()
	m.CreateThing(defaultDeviceID, td1Json)

	// retrieve a thing by ID
	td2Json, err := m.RetrieveThing(thingID)
	require.NoError(t, err)
	tdi2, err := td.UnmarshalTD(td2Json)
	assert.NoError(t, err)
	assert.Equal(t, thingID, tdi2.ID)
	assert.Equal(t, td1Json, td2Json)

	// delete a thing
	err = m.DeleteThing(defaultDeviceID, thingID)
	assert.NoError(t, err)
}

// Test using the messaging API to create and read things
func TestCRUDUsingMsgAPI(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	const clientID = "user1"

	testEnv, m, cancelFn := StartDirectoryServer(false)
	_ = testEnv
	defer cancelFn()

	directoryID := directory.DefaultDirectoryThingID
	thing1ID := clientID + ":thing1"

	// test create a TD
	tdi1 := td.NewTD(thing1ID, "thing 1", "device")
	tdi1Json := tdi1.ToString()

	// use a direct transport to the directory as the sink for the client
	tp := testenv.NewTestTransport(clientID, m)

	// err := dirClient.CreateThing(tdi1Json)
	err := directory_service.UpdateTD(directoryID, tdi1Json, tp.HandleRequest)
	require.NoError(t, err)

	// read the new TD
	dirTDD, _ := m.GetTDD()
	dirClient := directory_client.NewDirectoryClient(dirTDD, tp)
	tdi2, err := dirClient.RetrieveThing(thing1ID)
	require.NoError(t, err)
	assert.Equal(t, thing1ID, tdi2.ID)

	// delete the new TD
	err = dirClient.DeleteThing(thing1ID)
	require.NoError(t, err)

	// read should fail
	_, err = dirClient.RetrieveThing(thing1ID)
	require.Error(t, err)
}

// Get the directory TD on the http server well-known endpoint
func TestGetDirectoryTD(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	const userID = "user1"
	var dirTD *td.TD

	testEnv, m, cancelFn := StartDirectoryServer(true)
	defer cancelFn()
	assert.NotEmpty(t, m)

	httpURL := testEnv.HttpServer.GetConnectURL()
	parts, _ := url.Parse(httpURL)
	hostPort := parts.Host
	// Create a client account with token but don't use the client itself. This
	// tests is specifically for using a basic http client to bootstrap discovery.
	cl, token := testEnv.NewConnectedClient(userID, authn.ClientRoleViewer)
	cl.Close()
	// token, _, err := testEnv.CreateToken(userID, time.Minute)
	// require.NoError(t, err)

	httpClient := tls_client.NewTLSClient(hostPort, testEnv.CertBundle.CaCert, time.Minute)
	err := httpClient.AuthenticateWithToken(userID, token)
	require.NoError(t, err)
	defer httpClient.Close()

	respBody, statusCode, err := httpClient.Get(directory.WellKnownWoTPath)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, statusCode)

	err = jsoniter.Unmarshal(respBody, &dirTD)
	require.NoError(t, err)
	assert.Equal(t, directory.DefaultDirectoryThingID, dirTD.ID)
}

// Read the directory using the http api
func TestCRUDUsingRestAPI(t *testing.T) {
	t.Logf("---%s---\n", t.Name())

	const clientID = "device-1"
	thing1ID := "device-1:thing1"

	testEnv, m, cancelFn := StartDirectoryServer(true)
	defer cancelFn()
	assert.NotEmpty(t, m)

	// normally discovery provides the address
	dirTDD, dirTDDJson := m.GetTDD()
	require.NotEmpty(t, dirTDD)
	require.NotEmpty(t, dirTDDJson)

	// create the client account
	cl, authToken := testEnv.NewConnectedClient(clientID, authn.ClientRoleManager)
	cl.Close()
	// authToken, _, err := testEnv.CreateToken(clientID, time.Minute)
	// require.NoError(t, err)

	// test the http client
	dirClient := directory_client.NewDirectoryHttpClient(dirTDD, testEnv.CertBundle.CaCert)

	// FIXME: the http client should be able to do this using forms

	// connect should read the directory TD
	err := dirClient.AuthenticateWithToken(clientID, authToken)
	require.NoError(t, err)

	// test create a TD
	tdi1 := td.NewTD(thing1ID, "thing 1", "device")
	tdi1Json := tdi1.ToString()

	err = dirClient.CreateThing(tdi1Json)
	require.NoError(t, err)

	// read the new TD
	tdi2, err := dirClient.RetrieveThing(thing1ID)
	require.NoError(t, err)
	assert.Equal(t, thing1ID, tdi2.ID)

	// delete the new TD
	err = dirClient.DeleteThing(thing1ID)
	require.NoError(t, err)

	// read should fail
	slog.Error("---expect an error below---")
	_, err = dirClient.RetrieveThing(thing1ID)
	require.Error(t, err)
}

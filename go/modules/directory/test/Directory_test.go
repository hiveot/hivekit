package directory_test

import (
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules/authn"
	"github.com/hiveot/hivekit/go/modules/directory"
	directorypkg "github.com/hiveot/hivekit/go/modules/directory/pkg"
	"github.com/hiveot/hivekit/go/modules/transport"
	httptransportpkg "github.com/hiveot/hivekit/go/modules/transport/httptransport/pkg"
	"github.com/hiveot/hivekit/go/testenv"
	"github.com/hiveot/hivekit/go/utils"
	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var storageDir = filepath.Join(os.TempDir(), "hivekit", "directory-test")

const defaultAgentID = "agent-smith"
const defaultProtocol = transport.ProtocolTypeWotWebsocket
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
// withHttp means that directory service API is used for serving TDD and directory requests.
func StartDirectoryServer(withHttp bool) (
	testEnv *testenv.TestEnv, m directory.IDirectoryService, cancelFn func()) {

	var httpAPI transport.IHttpServer
	var dirHttpServer directory.IDirectoryHttpServer

	proto := defaultProtocol
	testEnv, cancelTestEnv := testenv.StartTestEnv(proto)
	transports := []transport.ITransportServer{testEnv.Server}

	if withHttp {
		// add directory endpoints to the http server
		dirHttpServer = directorypkg.NewDirectoryHttpServer(testEnv.HttpServer, rpcTimeout)
		transports = append(transports, dirHttpServer)
	}
	// the transports are used to update the TDD forms and security
	m = directorypkg.NewDirectoryMsgServer("", storageDir, httpAPI, transports)
	err := m.Start()
	if err != nil {
		panic("StartDirectoryServer: failed to start the directory " + err.Error())
	}
	if withHttp {
		dirHttpServer.SetRequestSink(m.HandleRequest)
	}

	// http requests are passed as RRN messages to the directory server
	// if httpAPI != nil {
	// httpAPI.SetRequestSink(m.HandleRequest)
	// }
	// RRN requests from the server are passed as RRN to the directory server
	testEnv.Server.SetRequestSink(m.HandleRequest)
	// the server receives the notifications
	m.SetNotificationSink(testEnv.Server.HandleNotification)
	return testEnv, m, func() {
		if httpAPI != nil {
			httpAPI.Stop()
		}
		m.Stop()
		cancelTestEnv()
	}
}

// Generic directory store testcases
func TestStartStop(t *testing.T) {
	t.Logf("---%s---\n", t.Name())

	m := directorypkg.NewDirectoryMsgServer("", storageDir, nil, nil)
	err := m.Start()
	require.NoError(t, err)
	defer m.Stop()

	// add a thing
	tdJson := directory.DirectoryTMJson
	m.UpdateThing(defaultAgentID, string(tdJson))

	// read all things
	tdList, err := m.RetrieveAllThings(0, 10)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(tdList), 1)
}

func TestCreateTD(t *testing.T) {
	thingID := "thing1"

	m := directorypkg.NewDirectoryMsgServer("", storageDir, nil, nil)
	err := m.Start()
	require.NoError(t, err)
	defer m.Stop()

	// add the directory itself
	tdJson := directory.DirectoryTMJson
	m.UpdateThing(defaultAgentID, string(tdJson))

	// read all things, expect 1
	tdList, err := m.RetrieveAllThings(0, 10)
	assert.NoError(t, err)
	assert.Len(t, tdList, 1)

	// add another TD
	tdi1 := td.NewTD(thingID, "test thing", "test device")
	td1Json := tdi1.ToString()
	m.CreateThing(defaultAgentID, td1Json)

	// retrieve a thing by ID
	td2Json, err := m.RetrieveThing(thingID)
	require.NoError(t, err)
	tdi2, err := td.UnmarshalTD(td2Json)
	assert.NoError(t, err)
	assert.Equal(t, thingID, tdi2.ID)
	assert.Equal(t, td1Json, td2Json)

	// delete a thing
	err = m.DeleteThing(defaultAgentID, thingID)
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
	err := directorypkg.UpdateTD(directoryID, tdi1Json, tp.HandleRequest)
	require.NoError(t, err)

	// read the new TD
	dirClient := directorypkg.NewDirectoryMsgClient(directoryID, tp)
	tdi2Json, err := dirClient.RetrieveThing(thing1ID)
	require.NoError(t, err)
	tdi2, err := td.UnmarshalTD(tdi2Json)
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

	// dirTM := m.GetTM()

	httpURL := testEnv.HttpServer.GetConnectURL()
	parts, _ := url.Parse(httpURL)
	hostPort := parts.Host
	// Create a client account with token but don't use the client itself. This
	// tests is specifically for using a basic http client to bootstrap discovery.
	cl, token := testEnv.NewConnectedClient(userID, authn.ClientRoleViewer)
	cl.Close()
	// token, _, err := testEnv.CreateToken(userID, time.Minute)
	// require.NoError(t, err)

	httpClient := httptransportpkg.NewHttpTransportClient(hostPort, testEnv.CertBundle.CaCert, time.Minute)
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

	const clientID = "agent1"
	thing1ID := "agent1:thing1"

	testEnv, m, cancelFn := StartDirectoryServer(true)
	defer cancelFn()
	assert.NotEmpty(t, m)

	// normally discovery provides the address
	dirTD, err := td.UnmarshalTD(m.RetrieveTDD())
	require.NoError(t, err)

	// create the client and connect to the http server that serves the directory TD
	cl, authToken := testEnv.NewConnectedClient(clientID, authn.ClientRoleManager)
	cl.Close()
	// authToken, _, err := testEnv.CreateToken(clientID, time.Minute)
	// require.NoError(t, err)

	dirClient := directorypkg.NewDirectoryHttpClient(dirTD, testEnv.CertBundle.CaCert)

	// connect should read the directory TD
	err = dirClient.AuthenticateWithToken(clientID, authToken)
	require.NoError(t, err)

	// test create a TD
	tdi1 := td.NewTD(thing1ID, "thing 1", "device")
	tdi1Json := tdi1.ToString()

	err = dirClient.CreateThing(tdi1Json)
	require.NoError(t, err)

	// read the new TD
	tdi2Json, err := dirClient.RetrieveThing(thing1ID)
	require.NoError(t, err)
	tdi2, err := td.UnmarshalTD(tdi2Json)
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

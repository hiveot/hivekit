package directory_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/hiveot/hivekit/go/modules/directory"
	directoryclient "github.com/hiveot/hivekit/go/modules/directory/client"
	"github.com/hiveot/hivekit/go/modules/directory/module"
	"github.com/hiveot/hivekit/go/modules/directory/server"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/modules/transports/direct"
	tlsclient "github.com/hiveot/hivekit/go/modules/transports/httpserver/client"
	"github.com/hiveot/hivekit/go/modules/transports/tptests"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hivekit/go/wot/td"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var storageRoot = ""

const defaultProtocol = transports.ProtocolTypeWotWSS
const TestKeyType = utils.KeyTypeED25519

// TestMain create a test folder for certificates and private key
func TestMain(m *testing.M) {
	utils.SetLogging("info", "")

	result := m.Run()
	if result != 0 {
		println("Test failed with code:", result)
	} else {
	}

	os.Exit(result)
}

// Start a test environment with a directory module connected to the server
func StartDirectoryServer() (
	testEnv *tptests.TestEnv, m directory.IDirectoryModule, cancelFn func()) {

	testEnv, cancelTestEnv := tptests.StartTestEnv(defaultProtocol)
	// use in-memory storage
	m = module.NewDirectoryModule(storageRoot, nil)
	err := m.Start("")
	if err != nil {
		panic("StartDirectoryServer: failed to start the directory " + err.Error())
	}
	// link the directory module to the server
	testEnv.Server.SetRequestSink(m.HandleRequest)
	m.SetNotificationSink(testEnv.Server.HandleNotification)
	return testEnv, m, func() {
		m.Stop()
		cancelTestEnv()
	}
}

// Generic directory store testcases
func TestStartStop(t *testing.T) {
	t.Logf("---%s---\n", t.Name())

	m := module.NewDirectoryModule(storageRoot, nil)
	err := m.Start("")
	require.NoError(t, err)
	defer m.Stop()

	// add a thing
	tdJson := server.DirectoryTMJson
	m.UpdateThing(string(tdJson))

	// read all things
	tdList, err := m.RetrieveAllThings(0, 10)
	assert.NoError(t, err)
	assert.NotNil(t, tdList)
}

func TestCreateTD(t *testing.T) {
	thingID := "thing1"

	m := module.NewDirectoryModule(storageRoot, nil)
	err := m.Start("")
	require.NoError(t, err)
	defer m.Stop()

	// add the directory itself
	tdJson := server.DirectoryTMJson
	m.UpdateThing(string(tdJson))

	// read all things, expect 1
	tdList, err := m.RetrieveAllThings(0, 10)
	assert.NoError(t, err)
	assert.Len(t, tdList, 1)

	// add another TD
	tdi1 := td.NewTD(thingID, "test thing", "test device")
	td1Json := tdi1.ToString()
	m.CreateThing(td1Json)

	// retrieve a thing by ID
	td2Json, err := m.RetrieveThing(thingID)
	require.NoError(t, err)
	tdi2, err := td.UnmarshalTD(td2Json)
	assert.NoError(t, err)
	assert.Equal(t, thingID, tdi2.ID)
	assert.Equal(t, td1Json, td2Json)

	// delete a thing
	err = m.DeleteThing(thingID)
	assert.NoError(t, err)
}

// Test using the messaging API to create and read things
func TestCRUDUsingMsgAPI(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	const clientID = "user1"

	testEnv, m, cancelFn := StartDirectoryServer()
	_ = testEnv
	defer cancelFn()

	directoryID := directory.DefaultDirectoryThingID
	thing1ID := "thing1"

	// test create a TD
	tdi1 := td.NewTD(thing1ID, "thing 1", "device")
	tdi1Json := tdi1.ToString()

	// use a direct transport to the directory as the sink for the client
	tp := direct.NewDirectTransport(clientID, m)
	dirClient := directoryclient.NewDirectoryMsgClient(directoryID, tp)
	err := dirClient.CreateThing(tdi1Json)
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
	_, err = dirClient.RetrieveThing(thing1ID)
	require.Error(t, err)
}

// Get the directory TD on the http server well-known endpoint
func TestDiscoverDirectory(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	const userID = "user1"
	var dirTD *td.TD

	testEnv, m, cancelFn := StartDirectoryServer()
	defer cancelFn()
	assert.NotEmpty(t, m)

	dirTM := m.GetTM()

	httpURL := testEnv.HttpServer.GetConnectURL()
	tdURL := fmt.Sprintf("%s%s", httpURL, directory.WellKnownWoTPath)
	httpClient := tlsclient.NewTLSClient(httpURL, nil, testEnv.CertBundle.CaCert, time.Minute)
	respBody, statusCode, err := httpClient.Get(tdURL)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, statusCode)

	err = json.Unmarshal(respBody, &dirTD)
	require.NoError(t, err)

	assert.Equal(t, dirTM, string(respBody))
}

// Read the directory using the http api
func TestCRUDUsingRestAPI(t *testing.T) {
	t.Logf("---%s---\n", t.Name())

	const clientID = "user1"

	thing1ID := "thing1"

	testEnv, cancelFn := tptests.StartTestEnv(defaultProtocol)
	defer cancelFn()

	// normally discovery provides the address
	tddUrl := testEnv.HttpServer.GetConnectURL()

	// create the client and connect to the http server that serves the directory TD
	err := testEnv.Authenticator.AddClient(clientID, transports.ClientRoleManager, "", "")
	require.NoError(t, err)
	authToken, _, err := testEnv.Authenticator.CreateToken(clientID, time.Minute)
	require.NoError(t, err)
	require.NoError(t, err)

	dirClient := directoryclient.NewDirectoryHttpClient(tddUrl, testEnv.CertBundle.CaCert)
	// connect should read the directory TD
	err = dirClient.ConnectWithToken(clientID, authToken)
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
	_, err = dirClient.RetrieveThing(thing1ID)
	require.Error(t, err)
}

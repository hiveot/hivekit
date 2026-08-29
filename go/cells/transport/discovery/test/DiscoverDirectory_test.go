package discovery_test

import (
	"testing"
	"time"

	"github.com/hiveot/hivekit/go/api"
	directory_service "github.com/hiveot/hivekit/go/cells/directory/service"
	discovery_client "github.com/hiveot/hivekit/go/cells/transport/discovery/client"
	discovery_server "github.com/hiveot/hivekit/go/cells/transport/discovery/server"
	"github.com/hiveot/hivekit/go/testenv"
	"github.com/hiveot/hivekit/go/utils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// serviceID is the service publishing the record, thing or directory
const testDirServiceName = "hiveot-test"

// Test the directory discovery
func TestDiscoverDirectory(t *testing.T) {
	dirTdd := "{}"

	testServiceAddress := utils.GetOutboundIP("").String()
	endpoints := map[string]string{"wss": "wss://localhost/wssendpoint"}

	testEnv := testenv.NewTestEnv(true)
	testEnv.StartHttpServer(true)
	defer testEnv.HttpServer.Stop()

	m := discovery_server.NewDiscoveryServer(testDirServiceName, testEnv.HttpServer, "", endpoints)
	err := m.Start()
	require.NoError(t, err)
	defer m.Stop()

	err = m.ServeDirectoryTD(testDirServiceName, dirTdd)
	require.NoError(t, err)

	// Test if it is discovered on startup
	cl := discovery_client.NewDiscoveryClient(nil, true)
	err = cl.Start()
	assert.NoError(t, err)

	// records, err := cl.DiscoverDirectories(testServiceID, time.Second, true, nil)
	// rec0, err := cl.DiscoverFirstDirectory(testDirServiceName, time.Second)
	recs, err := cl.DiscoverDirectories(time.Second*1, nil)
	_ = recs
	rec0, err := cl.DiscoverFirstDirectory(testDirServiceName, time.Second)
	require.NoError(t, err)
	require.NotEmpty(t, rec0)
	assert.Equal(t, testDirServiceName, rec0.Instance)
	assert.Equal(t, testServiceAddress, rec0.Addr)
	assert.NotEmpty(t, rec0.TD)
	assert.Equal(t, true, rec0.IsDirectory)

	time.Sleep(time.Millisecond) // prevent race error in server
}

func TestDiscoverGetDirectoryTD(t *testing.T) {

	// the http server is needed to expose the TDD
	testEnv := testenv.NewTestEnv(true)
	testHttpServer, httpServerURL := testEnv.StartHttpServer(true)
	_ = httpServerURL
	defer testEnv.HttpServer.Stop()

	// the transport server for reading the directory
	// This is needed to set the connection information in the directory TDD.
	tpServer := testEnv.StartTestServer("")
	defer tpServer.Stop()

	// run a directory that will be discoverable
	tpList := []api.ITransportServer{tpServer}
	dirMod := directory_service.NewDirectoryService("", "", testHttpServer, tpList)
	dirMod.Start()
	dirThingID := dirMod.GetThingID()
	dirTD, dirTDJson := dirMod.GetTDD()
	_ = dirTD

	// dirTD := dirMod.GetTD(dirMod.GetThingID())
	// dirTDJson := td.MarshalTD(dirTD)

	// run the discover server and expose the directory TDD
	m := discovery_server.NewDiscoveryServer(testDirServiceName, testEnv.HttpServer, "", nil)
	err := m.Start()
	require.NoError(t, err)
	defer m.Stop()
	err = m.ServeDirectoryTD(testDirServiceName, dirTDJson)
	require.NoError(t, err)

	// discover and read the directory on start
	appEnv := api.NewHiveEnvironment("", false)
	cl := discovery_client.NewDiscoveryClient(appEnv, true)
	err = cl.Start()
	require.NoError(t, err)
	assert.NotEmpty(t, appEnv.DirectoryURL)
	dirTD2, _, err := cl.DiscoverFirstDirectoryTD(dirThingID, time.Second)
	require.NoError(t, err)
	assert.NotNil(t, dirTD2, "Client failed to discover the directory on start")
	assert.Equal(t, dirMod.GetThingID(), dirTD2.ID)
}

func TestDiscoverNoDirectory(t *testing.T) {
	// run the server
	// run the server
	testEnv := testenv.NewTestEnv(true)
	testHttpServer, httpServerURL := testEnv.StartHttpServer(true)
	_ = httpServerURL
	defer testEnv.HttpServer.Stop()

	// start discovery client
	cl := discovery_client.NewDiscoveryClient(testEnv.AppEnv, true)
	err := cl.Start()
	require.NoError(t, err)
	dirTD2, _, err := cl.DiscoverFirstDirectoryTD(testDirServiceName, time.Second)
	assert.Nil(t, dirTD2)

	// run the discover server without exposing the directory TDD
	m := discovery_server.NewDiscoveryServer(testDirServiceName, testHttpServer, "", nil)
	err = m.Start()
	require.NoError(t, err)
	defer m.Stop()
	err = m.ServeDirectoryTD(testDirServiceName, "") // empty json
	require.NoError(t, err)

	// restart discovery client
	cl.Stop()
	err = cl.Start()
	require.NoError(t, err)

	// no directory has been found
	dirTD2, _, err = cl.DiscoverFirstDirectoryTD(testDirServiceName, time.Second)
	require.Error(t, err)
	assert.Nil(t, dirTD2)
}

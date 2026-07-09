package discovery_test

import (
	"testing"
	"time"

	"github.com/hiveot/hivekit/go/api"
	directorypkg "github.com/hiveot/hivekit/go/modules/directory/pkg"
	discoverypkg "github.com/hiveot/hivekit/go/modules/transport/discovery/pkg"
	"github.com/hiveot/hivekit/go/testenv"
	"github.com/hiveot/hivekit/go/utils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// serviceID is the service publishing the record, thing or directory
const testDirServiceID = "hiveot-test"
const testDirServicePort = 9999

// Test the discovery without using the module
func TestDiscoverDirectory(t *testing.T) {
	dirTdd := "{}"

	testServiceAddress := utils.GetOutboundIP("").String()
	endpoints := map[string]string{"wss": "wss://localhost/wssendpoint"}

	testEnv := testenv.NewTestEnv(true)
	testEnv.StartHttpServer(true)
	defer testEnv.HttpServer.Stop()

	m := discoverypkg.NewDirectoryDiscoveryServer(testDirServiceID, testEnv.HttpServer, endpoints)
	err := m.Start()
	require.NoError(t, err)
	defer m.Stop()

	err = m.ServeDirectoryTD(dirTdd)
	require.NoError(t, err)

	// Test if it is discovered on startup
	cl := discoverypkg.NewDiscoveryClient(nil, true)
	err = cl.Start()
	assert.NoError(t, err)

	// records, err := cl.DiscoverDirectories(testServiceID, time.Second, true, nil)
	rec0, err := cl.DiscoverFirstDirectory(testDirServiceID, time.Second)
	require.NoError(t, err)
	require.NotEmpty(t, rec0)
	assert.Equal(t, testDirServiceID, rec0.Instance)
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
	dirMod := directorypkg.NewDirectoryService("", "", testHttpServer, tpList)
	dirMod.Start()
	_, dirTDJson := dirMod.GetTDD()
	// dirTD := dirMod.GetTD(dirMod.GetThingID())
	// dirTDJson := td.MarshalTD(dirTD)

	// run the discover server and expose the directory TDD
	m := discoverypkg.NewDirectoryDiscoveryServer(testDirServiceID, testEnv.HttpServer, nil)
	err := m.Start()
	require.NoError(t, err)
	defer m.Stop()
	err = m.ServeDirectoryTD(dirTDJson)
	require.NoError(t, err)

	// discover and read the directory on start
	appEnv := api.NewAppEnvironment("", false)
	cl := discoverypkg.NewDiscoveryClient(appEnv, true)
	err = cl.Start()
	require.NoError(t, err)
	assert.NotEmpty(t, appEnv.DirectoryURL)

	dirTD2, _, err := cl.DiscoverFirstDirectoryTD(testDirServiceID, time.Second)
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
	cl := discoverypkg.NewDiscoveryClient(testEnv.AppEnv, true)
	err := cl.Start()
	require.NoError(t, err)
	dirTD2, _, err := cl.DiscoverFirstDirectoryTD(testDirServiceID, time.Second)
	assert.Nil(t, dirTD2)

	// run the discover server without exposing the directory TDD
	m := discoverypkg.NewDirectoryDiscoveryServer(testDirServiceID, testHttpServer, nil)
	err = m.Start()
	require.NoError(t, err)
	defer m.Stop()
	err = m.ServeDirectoryTD("") // empty json
	require.NoError(t, err)

	// restart discovery client
	cl.Stop()
	err = cl.Start()
	require.NoError(t, err)

	// no directory has been found
	dirTD2, _, err = cl.DiscoverFirstDirectoryTD(testDirServiceID, time.Second)
	require.Error(t, err)
	assert.Nil(t, dirTD2)
}

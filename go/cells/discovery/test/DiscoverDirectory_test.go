package discovery_test

import (
	"testing"
	"time"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/api/vocab"
	directory_service "github.com/hiveot/hivekit/go/cells/directory/service"
	"github.com/hiveot/hivekit/go/cells/discovery"
	discovery_client "github.com/hiveot/hivekit/go/cells/discovery/client"
	discovery_server "github.com/hiveot/hivekit/go/cells/discovery/server"
	"github.com/hiveot/hivekit/go/testenv"
	"github.com/hiveot/hivekit/go/utils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// serviceID is the service publishing the record, thing or directory
const testDirServiceName = "hiveot-test"

// Test the directory discovery
func TestDiscoverDirectory(t *testing.T) {

	dirTDD := td.NewTD("tddID", "testTDD", vocab.DeviceTypeService)

	testServiceAddress := utils.GetOutboundIP("").String()
	endpoints := map[string]string{"wss": "wss://localhost/wssendpoint"}

	testEnv := testenv.NewTestEnv(true)
	testEnv.StartHttpServer(true)
	defer testEnv.Stop()

	discoSrv, err := discovery_server.NewDiscoveryServer(testEnv.HttpServer, nil, endpoints, nil)
	require.NoError(t, err)
	defer discoSrv.Stop()

	tddURL, err := discoSrv.ServeDirectoryTD(testDirServiceName, dirTDD)
	require.NoError(t, err)
	assert.NotEmpty(t, tddURL)

	// Test if it is discovered on startup
	cl, err := discovery_client.NewDiscoveryClient(nil, true)
	assert.NoError(t, err)

	// records, err := cl.DiscoverDirectories(testServiceID, time.Second, true, nil)
	// rec0, err := cl.DiscoverFirstDirectory(testDirServiceName, time.Second)
	recs, err := cl.DiscoverThings("", discovery.DISCO_TYPE_DIRECTORY, false, time.Second*1, nil)
	require.NoError(t, err)
	_ = recs
	// rec0, err := cl.DiscoverFirstDirectory(testDirServiceName, time.Second)
	rec0 := cl.DiscoverFirstThing(testDirServiceName, discovery.DISCO_TYPE_DIRECTORY, time.Second*1)
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
	_ = testHttpServer
	defer testEnv.Stop()

	// the transport server for reading the directory
	// This is needed to set the connection information in the directory TDD.
	tpServer := testEnv.StartTestServer("")
	defer tpServer.Stop()

	// run a directory that will be discoverable
	dirSvc, err := directory_service.NewDirectoryService("", "", testEnv.AddForms)
	dirTD := dirSvc.GetTDD()

	// run the discover server and expose the directory TDD
	discoSvc, err := discovery_server.NewDiscoveryServer(
		testEnv.HttpServer, nil, nil, testEnv.AddForms)
	require.NoError(t, err)
	defer discoSvc.Stop()
	// testDirServiceName is appended to the well-known path
	tddURL, err := discoSvc.ServeDirectoryTD(testDirServiceName, dirTD)
	require.NoError(t, err)

	// download the TDD from the well known endpoint
	// httpClient := tls_client.NewTLSClient(httpServerURL, testEnv.CertBundle.RootCAs)
	// httpClient.SetTimeout(testEnv.Env.RpcTimeout)
	// defer httpClient.Close()
	// respBody, statusCode, err := httpClient.Get(discovery.WellKnownHttpPath)
	// require.NoError(t, err)
	// require.Equal(t, http.StatusOK, statusCode)
	// err = jsoniter.Unmarshal(respBody, &dirTD)
	// require.NoError(t, err)
	// assert.Equal(t, dirSvc.GetID(), dirTD.ID)

	// discover and read the directory on start. This sets env.DirectoryURL
	appEnv := api.NewHiveEnvironment("", false)
	appEnv.ServerTDURL = tddURL
	cl, err := discovery_client.NewDiscoveryClient(appEnv, true)
	require.NoError(t, err)
	// the discovery server can publish multiple records
	assert.NotEmpty(t, appEnv.ServerTDURL)

	dirTD2 := cl.DiscoverFirstTD(
		testDirServiceName, discovery.DISCO_TYPE_DIRECTORY, time.Second)
	require.NoError(t, err)
	require.NotNil(t, dirTD2, "Client failed to discover the directory on start")
	assert.Equal(t, tddURL, appEnv.ServerTDURL)
	assert.Equal(t, dirSvc.GetID(), dirTD2.ID)
}

func TestDiscoverNoDirectory(t *testing.T) {
	// run the server
	// run the server
	testEnv := testenv.NewTestEnv(true)
	testHttpServer, httpServerURL := testEnv.StartHttpServer(true)
	_ = httpServerURL
	defer testEnv.Stop()

	// start discovery client
	cl, err := discovery_client.NewDiscoveryClient(testEnv.Env, true)
	require.NoError(t, err)
	dirTD2 := cl.DiscoverFirstTD(
		testDirServiceName, discovery.DISCO_TYPE_DIRECTORY, time.Second)
	assert.Nil(t, dirTD2)

	// run the discover server without exposing the directory TDD
	discoSrv, err := discovery_server.NewDiscoveryServer(
		testHttpServer, nil, nil, testEnv.AddForms)
	require.NoError(t, err)
	defer discoSrv.Stop()
	tddURL, err := discoSrv.ServeDirectoryTD(testDirServiceName, nil) // empty json
	assert.Empty(t, tddURL)
	require.Error(t, err)

}

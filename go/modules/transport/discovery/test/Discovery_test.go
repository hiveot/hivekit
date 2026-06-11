package discovery_test

import (
	"net/url"
	"testing"
	"time"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules/directory"
	discoverypkg "github.com/hiveot/hivekit/go/modules/transport/discovery/pkg"
	"github.com/hiveot/hivekit/go/testenv"
	"github.com/hiveot/hivekit/go/utils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// serviceID is the service publishing the record, thing or directory
const testServiceID = "hiveot-test"
const testServicePort = 9999

// Test the discovery without using the module
func TestDiscoverDirectory(t *testing.T) {
	dirTdd := "{this is a directory TD}"

	testServiceAddress := utils.GetOutboundIP("").String()
	endpoints := map[string]string{"wss": "wss://localhost/wssendpoint"}

	testEnv := testenv.NewTestEnv()
	testEnv.StartHttpServer(true)
	defer testEnv.HttpServer.Stop()

	m := discoverypkg.NewDiscoveryServer(testServiceID, testEnv.HttpServer, endpoints)
	err := m.Start()
	require.NoError(t, err)
	defer m.Stop()

	err = m.ServeDirectoryTD(dirTdd)
	require.NoError(t, err)

	// Test if it is discovered
	cl := discoverypkg.NewDiscoveryClient()
	// records, err := cl.DiscoverDirectories(testServiceID, time.Second, true, nil)
	records, err := cl.DiscoverDirectories(testServiceID, time.Second, true, nil)
	require.NoError(t, err)
	require.NotEmpty(t, records)
	rec0 := records[0]
	assert.Equal(t, testServiceID, rec0.Instance)
	assert.Equal(t, testServiceAddress, rec0.Addr)
	assert.NotEmpty(t, rec0.TD)
	assert.Equal(t, true, rec0.IsDirectory)

	time.Sleep(time.Millisecond) // prevent race error in server
}

// discover things without the server module
func TestDiscoverThings(t *testing.T) {
	// thingTD := "this is a Thing TD"

	testEnv := testenv.NewTestEnv()
	testEnv.StartHttpServer(true)
	defer testEnv.HttpServer.Stop()

	m := discoverypkg.NewDiscoveryServer(testServiceID, testEnv.HttpServer, nil)
	err := m.Start()
	require.NoError(t, err)
	defer m.Stop()
	err = m.ServeThingTD(testServiceID)
	require.NoError(t, err)

	// Test if it is discovered
	serverAddr := testEnv.HttpServer.GetConnectURL()
	urlParts, _ := url.Parse(serverAddr)
	cl := discoverypkg.NewDiscoveryClient()
	records, err := cl.DiscoverThings(testServiceID, time.Second, nil)
	require.NoError(t, err)
	require.Equal(t, len(records), 1, "the test thing record was not discovered")
	rec0 := records[0]
	assert.Equal(t, urlParts.Hostname(), rec0.Addr)
}

func TestDiscoverGetDirectoryTD(t *testing.T) {
	dirTDJson := "this is the test JSON"

	// run the server
	testEnv := testenv.NewTestEnv()
	testEnv.StartHttpServer(true)
	defer testEnv.HttpServer.Stop()
	m := discoverypkg.NewDiscoveryServer(testServiceID, testEnv.HttpServer, nil)
	err := m.Start()
	require.NoError(t, err)
	defer m.Stop()
	err = m.ServeDirectoryTD(dirTDJson)
	require.NoError(t, err)

	// discover the server
	cl := discoverypkg.NewDiscoveryClient()
	record, err := cl.DiscoverFirstDirectory(testServiceID)
	require.NoError(t, err)
	require.NotEmpty(t, record)
	tddJSON, err := cl.DownloadTDD(record.AsURL(), nil)
	assert.NoError(t, err)
	assert.True(t, record.IsDirectory)
	assert.Equal(t, len(dirTDJson), len(tddJSON))
}

func TestDiscoverGetThingTD(t *testing.T) {

	// run the server
	testEnv := testenv.NewTestEnv()
	testEnv.StartHttpServer(true)
	defer testEnv.HttpServer.Stop()
	thingTD := testEnv.CreateTestTD(12)

	m := discoverypkg.NewDiscoveryServer(testServiceID, testEnv.HttpServer, nil)
	err := m.Start()
	require.NoError(t, err)
	defer m.Stop()

	// publish a TD in the module chain
	// err = m.ServeThingTD(thingTD)
	tdJson1, _ := td.MarshalTD(thingTD)
	req := msg.NewRequestMessage(td.OpInvokeAction,
		directory.DefaultDirectoryThingID, directory.CreateThingAction, tdJson1)
	err = m.HandleRequest(req, nil)
	require.NoError(t, err)

	// discover the server
	cl := discoverypkg.NewDiscoveryClient()
	records, err := cl.DiscoverThings(testServiceID, time.Second, nil)
	require.NoError(t, err)
	require.NotZero(t, len(records), "no things discovered")
	tdJson2, err := cl.DownloadTDD(records[0].AsURL(), nil)
	assert.NoError(t, err)
	assert.True(t, records[0].IsThing)
	assert.Equal(t, tdJson1, tdJson2)
}

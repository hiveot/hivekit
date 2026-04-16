package discovery_test

import (
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/grandcat/zeroconf"
	"github.com/hiveot/hivekit/go/modules/transports/discovery"
	discoveryclient "github.com/hiveot/hivekit/go/modules/transports/discovery/client"
	"github.com/hiveot/hivekit/go/modules/transports/discovery/internal"
	"github.com/hiveot/hivekit/go/testenv"
	"github.com/hiveot/hivekit/go/utils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// serviceID is the service publishing the record, thing or directory
const testServiceID = "hiveot-test"
const testServicePort = 9999

func TestDNSSDScan(t *testing.T) {
	var count atomic.Int32

	r, err := discoveryclient.DnsSDScan("", "", time.Second*2,
		func(_ *zeroconf.ServiceEntry) bool {
			count.Add(1)
			return false
		})

	nrRecords := int(count.Load())
	t.Logf("Found %d records in scan", nrRecords)

	assert.NoError(t, err)
	assert.Equal(t, nrRecords, len(r))
	assert.Greater(t, nrRecords, 0, "No DNS records found")
}

func TestDiscover(t *testing.T) {
	testServiceType := "_discovery.test-type._tcp"
	address := utils.GetOutboundIP("").String()

	srv, err := internal.ServeDnsSD(
		testServiceID, testServiceType, address, testServicePort, nil)
	assert.NoError(t, err)
	defer srv.Shutdown()

	r, err := discoveryclient.DnsSDScan(testServiceID, testServiceType, time.Second,
		func(*zeroconf.ServiceEntry) bool {
			return true // stop
		})
	require.Len(t, r, 1)
	assert.Equal(t, testServiceID, r[0].Instance)
}

func TestNoInstanceID(t *testing.T) {
	address := utils.GetOutboundIP("").String()
	testServiceType := "test-service-type"

	_, err := internal.ServeDnsSD(
		"", testServiceType, address, testServicePort, nil)
	assert.Error(t, err) // missing instance name

	_, err = internal.ServeDnsSD(
		testServiceID, "", address, testServicePort, nil)
	assert.Error(t, err) // missing service name
}

func TestBadAddress(t *testing.T) {
	testServiceType := "test-service-type"

	discoServer, err := internal.ServeDnsSD(
		testServiceID, testServiceType, "notanipaddress", testServicePort, nil)

	assert.Error(t, err)
	assert.Nil(t, discoServer)
}

func TestExternalAddress(t *testing.T) {
	testServiceType := "test-service-type"

	discoServer, err := internal.ServeDnsSD(
		testServiceID, testServiceType, "1.2.3.4", testServicePort, nil)

	// expect a warning
	assert.NoError(t, err)
	time.Sleep(time.Millisecond) // prevent race error with server
	discoServer.Shutdown()
}

// Test the discovery without using the module
func TestDiscoverDirectory(t *testing.T) {
	dirTdd := "{this is a directory TD}"

	testServiceAddress := utils.GetOutboundIP("").String()
	endpoints := map[string]string{"wss": "wss://localhost/wssendpoint"}

	testEnv := testenv.NewTestEnv()
	testEnv.StartHttpServer()
	defer testEnv.HttpServer.Stop()

	m := discovery.NewDiscoveryServer(testEnv.HttpServer, endpoints, testServiceID)
	err := m.Start()
	require.NoError(t, err)
	defer m.Stop()

	err = m.ServeDirectoryTDD(dirTdd)
	require.NoError(t, err)

	// Test if it is discovered
	cl := discoveryclient.NewDiscoveryClient()
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
	testEnv.StartHttpServer()
	defer testEnv.HttpServer.Stop()

	m := discovery.NewDiscoveryServer(testEnv.HttpServer, nil, testServiceID)
	err := m.Start()
	require.NoError(t, err)
	defer m.Stop()
	err = m.ServeThingTD(testServiceID)
	require.NoError(t, err)

	// Test if it is discovered
	serverAddr := testEnv.HttpServer.GetConnectURL()
	urlParts, _ := url.Parse(serverAddr)
	cl := discoveryclient.NewDiscoveryClient()
	records, err := cl.DiscoverThings(testServiceID, time.Second, nil)
	require.NoError(t, err)
	require.Equal(t, len(records), 1, "the test thing record was not discovered")
	rec0 := records[0]
	assert.Equal(t, urlParts.Hostname(), rec0.Addr)
}

func TestDiscoverBadPort(t *testing.T) {
	testServiceType := "test-service-type"

	badPort := 0
	address := utils.GetOutboundIP("").String()
	_, err := internal.ServeDnsSD(testServiceID, testServiceType, address, badPort, nil)

	assert.Error(t, err)
}

func TestDiscoverGetDirectoryTD(t *testing.T) {
	dirTDJson := "this is the test JSON"

	// run the server
	testEnv := testenv.NewTestEnv()
	testEnv.StartHttpServer()
	defer testEnv.HttpServer.Stop()
	m := discovery.NewDiscoveryServer(testEnv.HttpServer, nil, testServiceID)
	err := m.Start()
	require.NoError(t, err)
	defer m.Stop()
	err = m.ServeDirectoryTDD(dirTDJson)
	require.NoError(t, err)

	// discover the server
	cl := discoveryclient.NewDiscoveryClient()
	record, err := cl.DiscoverFirstDirectory(testServiceID)
	require.NoError(t, err)
	require.NotEmpty(t, record)
	tddJSON, err := cl.DownloadTDD(record.AsURL(), nil)
	assert.NoError(t, err)
	assert.True(t, record.IsDirectory)
	assert.Equal(t, len(dirTDJson), len(tddJSON))
}

func TestDiscoverGetThingTD(t *testing.T) {
	thingTD := "{this is the test thing TD}"

	// run the server
	testEnv := testenv.NewTestEnv()
	testEnv.StartHttpServer()
	defer testEnv.HttpServer.Stop()
	m := discovery.NewDiscoveryServer(testEnv.HttpServer, nil, testServiceID)
	err := m.Start()
	require.NoError(t, err)
	defer m.Stop()
	err = m.ServeThingTD(thingTD)
	require.NoError(t, err)

	// discover the server
	cl := discoveryclient.NewDiscoveryClient()
	records, err := cl.DiscoverThings(testServiceID, time.Second, nil)
	require.NoError(t, err)
	require.NotZero(t, len(records), "no things discovered")
	tdJSON, err := cl.DownloadTDD(records[0].AsURL(), nil)
	assert.NoError(t, err)
	assert.True(t, records[0].IsDirectory)
	assert.Equal(t, thingTD, tdJSON)
}

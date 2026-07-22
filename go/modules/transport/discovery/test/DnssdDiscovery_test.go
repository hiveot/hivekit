package discovery_test

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/grandcat/zeroconf"

	"github.com/hiveot/hivekit/go/modules/transport/discovery/internal/clientimpl"
	"github.com/hiveot/hivekit/go/modules/transport/discovery/internal/serverimpl"
	"github.com/hiveot/hivekit/go/utils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// serviceID is the service publishing the record, thing or directory
// const testServiceID = "hiveot-test"
// const testServicePort = 9999

func TestDNSSDScan(t *testing.T) {
	var count atomic.Int32

	r, err := clientimpl.DnsSDScan("", "", time.Second*2,
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

	srv, err := serverimpl.ServeDnsSD(
		testServiceID, testServiceType, address, testServicePort, nil)
	assert.NoError(t, err)
	defer srv.Shutdown()

	r, err := clientimpl.DnsSDScan(testServiceID, testServiceType, time.Second,
		func(*zeroconf.ServiceEntry) bool {
			return true // stop
		})
	require.Len(t, r, 1)
	assert.Equal(t, testServiceID, r[0].Instance)
}

func TestNoInstanceID(t *testing.T) {
	address := utils.GetOutboundIP("").String()
	testServiceType := "test-service-type"

	_, err := serverimpl.ServeDnsSD(
		"", testServiceType, address, testServicePort, nil)
	assert.Error(t, err) // missing instance name

	_, err = serverimpl.ServeDnsSD(
		testServiceID, "", address, testServicePort, nil)
	assert.Error(t, err) // missing service name
}

func TestBadAddress(t *testing.T) {
	testServiceType := "test-service-type"

	discoServer, err := serverimpl.ServeDnsSD(
		testServiceID, testServiceType, "notanipaddress", testServicePort, nil)

	assert.Error(t, err)
	assert.Nil(t, discoServer)
}

func TestExternalAddress(t *testing.T) {
	testServiceType := "test-service-type"

	discoServer, err := serverimpl.ServeDnsSD(
		testServiceID, testServiceType, "1.2.3.4", testServicePort, nil)

	// expect a warning
	assert.NoError(t, err)
	time.Sleep(time.Millisecond) // prevent race error with server
	discoServer.Shutdown()
}

func TestDiscoverBadPort(t *testing.T) {
	testServiceType := "test-service-type"

	badPort := 0
	address := utils.GetOutboundIP("").String()
	_, err := serverimpl.ServeDnsSD(testServiceID, testServiceType, address, badPort, nil)

	assert.Error(t, err)
}

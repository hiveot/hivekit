package discovery_test

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/grandcat/zeroconf"

	internal "github.com/hiveot/hivekit/go/modules/transport/discovery/internal/client"
	internalserver "github.com/hiveot/hivekit/go/modules/transport/discovery/internal/server"
	"github.com/hiveot/hivekit/go/utils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// serviceID is the service publishing the record, thing or directory
// const testServiceID = "hiveot-test"
// const testServicePort = 9999

func TestDNSSDScan(t *testing.T) {
	var count atomic.Int32

	r, err := internal.DnsSDScan("", "", time.Second*2,
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

	srv, err := internalserver.ServeDnsSD(
		testThingServiceID, testServiceType, address, testThingServicePort, nil)
	assert.NoError(t, err)
	defer srv.Shutdown()

	r, err := internal.DnsSDScan(testThingServiceID, testServiceType, time.Second,
		func(*zeroconf.ServiceEntry) bool {
			return true // stop
		})
	require.Len(t, r, 1)
	assert.Equal(t, testThingServiceID, r[0].Instance)
}

func TestNoInstanceID(t *testing.T) {
	address := utils.GetOutboundIP("").String()
	testServiceType := "test-service-type"

	_, err := internalserver.ServeDnsSD(
		"", testServiceType, address, testThingServicePort, nil)
	assert.Error(t, err) // missing instance name

	_, err = internalserver.ServeDnsSD(
		testThingServiceID, "", address, testThingServicePort, nil)
	assert.Error(t, err) // missing service name
}

func TestBadAddress(t *testing.T) {
	testServiceType := "test-service-type"

	discoServer, err := internalserver.ServeDnsSD(
		testThingServiceID, testServiceType, "notanipaddress", testThingServicePort, nil)

	assert.Error(t, err)
	assert.Nil(t, discoServer)
}

func TestExternalAddress(t *testing.T) {
	testServiceType := "test-service-type"

	discoServer, err := internalserver.ServeDnsSD(
		testThingServiceID, testServiceType, "1.2.3.4", testThingServicePort, nil)

	// expect a warning
	assert.NoError(t, err)
	time.Sleep(time.Millisecond) // prevent race error with server
	discoServer.Shutdown()
}

func TestDiscoverBadPort(t *testing.T) {
	testServiceType := "test-service-type"

	badPort := 0
	address := utils.GetOutboundIP("").String()
	_, err := internalserver.ServeDnsSD(testThingServiceID, testServiceType, address, badPort, nil)

	assert.Error(t, err)
}

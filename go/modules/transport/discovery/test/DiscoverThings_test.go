package discovery_test

import (
	"net/url"
	"testing"
	"time"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules/directory"
	discoverypkg "github.com/hiveot/hivekit/go/modules/transport/discovery/pkg"
	"github.com/hiveot/hivekit/go/testenv"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// filter on test serviceID to avoid interference with running services
const testThingServiceID = "hiveot-test"
const testThingServicePort = 9999

// discover things without the server module
func TestDiscoverThings(t *testing.T) {
	// thingTD := "this is a Thing TD"

	testEnv := testenv.NewTestEnv(true)
	testEnv.StartHttpServer(true)
	defer testEnv.HttpServer.Stop()

	m := discoverypkg.NewDiscoveryServer(testThingServiceID, testEnv.HttpServer, nil)
	err := m.Start()
	require.NoError(t, err)
	defer m.Stop()
	err = m.ServeThingTD(testThingServiceID)
	require.NoError(t, err)

	// Test if it is discovered
	serverAddr := testEnv.HttpServer.GetConnectURL()
	urlParts, _ := url.Parse(serverAddr)
	cl := discoverypkg.NewDiscoveryClient(nil, false)
	records, err := cl.DiscoverThings(testThingServiceID, time.Second, nil)
	require.NoError(t, err)
	require.Equal(t, len(records), 1, "the test thing record was not discovered")
	rec0 := records[0]
	assert.Equal(t, urlParts.Hostname(), rec0.Addr)
}

func TestDiscoverGetThingTD(t *testing.T) {

	// run the server
	testEnv := testenv.NewTestEnv(true)
	testEnv.StartHttpServer(true)
	defer testEnv.HttpServer.Stop()
	thingTD := testEnv.CreateTestTD(12)

	m := discoverypkg.NewDiscoveryServer(testThingServiceID, testEnv.HttpServer, nil)
	err := m.Start()
	require.NoError(t, err)
	defer m.Stop()

	// publish a TD in the module chain.
	// This should be handled by the discovery server.
	// err = m.ServeThingTD(thingTD)
	tdJson1 := td.MarshalTD(thingTD)
	req := msg.NewRequestMessage(td.OpInvokeAction,
		directory.DefaultDirectoryThingID, directory.CreateThingAction, tdJson1)
	err = m.HandleRequest(req, req.NoReply)
	require.NoError(t, err)

	// discover the server
	appEnv := api.NewAppEnvironment("", false)
	cl := discoverypkg.NewDiscoveryClient(appEnv, false)
	rec0, err := cl.DiscoverFirstGateway(testThingServiceID, time.Second)
	// records, err := cl.DiscoverThings(testThingServiceID, time.Second, nil)
	require.NoError(t, err)
	assert.True(t, rec0.IsThing)

	// require.NotZero(t, len(records), "no things discovered")
	td2, tdJson2, err := cl.LoadTD(rec0.AsURL(), nil)
	assert.NoError(t, err)
	assert.Equal(t, tdJson1, tdJson2)
	assert.Equal(t, thingTD.ID, td2.ID)
	// assert.Equal(t, cl.GetDirectoryURL(), appEnv.ServerURL)
}

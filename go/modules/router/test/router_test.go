package router_test

import (
	"fmt"
	"os"
	"testing"

	certstest "github.com/hiveot/hivekit/go/modules/certs/test"
	"github.com/hiveot/hivekit/go/modules/router"
	httpserverapi "github.com/hiveot/hivekit/go/modules/transports/httpserver/api"
	"github.com/hiveot/hivekit/go/modules/transports/tptests"
	"github.com/hiveot/hivekit/go/msg"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hivekit/go/wot"
	"github.com/hiveot/hivekit/go/wot/td"
	"github.com/stretchr/testify/require"
)

var testDevicePort = 9993
var testDeviceAddress = fmt.Sprintf(":%d", testDevicePort)
var certsBundle = certstest.CreateTestCertBundle(utils.KeyTypeED25519)
var testAuthn = tptests.NewTestAuthenticator()

const testDeviceID = "device1"

var testTD *td.TD

func getTD(thingID string) *td.TD {
	return testTD
}

func startVirtualDevice() (v *VirtualDevice) {
	// create a test device with server
	cfg := httpserverapi.NewConfig(
		certsBundle.ServerAddr, testDevicePort,
		certsBundle.ServerCert, certsBundle.CaCert, testAuthn.ValidateToken)

	v = NewVirtualDevice(cfg, testDeviceID)
	err := v.Start("")
	if err != nil {
		panic("failed starting test device")
	}
	return v
}

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

// Generic directory store testcases
func TestStartStop(t *testing.T) {
	t.Logf("---%s---\n", t.Name())

	m := router.NewRouterModule(getTD, nil)
	err := m.Start("")
	require.NoError(t, err)
	defer m.Stop()
}

// connect to a virtual device and subscribe to events
func TestSubscribeToDevice(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	const thingID1 = "thing-1"

	v := startVirtualDevice()
	defer v.Stop()
	req := msg.NewRequestMessage(wot.OpObserveAllProperties, thingID1, "", nil, "")
	v.HandleRequest(req, func(resp *msg.ResponseMessage) error {
		return nil
	})

	m := router.NewRouterModule(getTD, nil)
	err := m.Start("")
	require.NoError(t, err)
	defer m.Stop()
}

package vcache_test

import (
	"os"
	"testing"

	vcachemodule "github.com/hiveot/hivekit/go/modules/vcache/module"
	"github.com/hiveot/hivekit/go/msg"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hivekit/go/wot"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

	m := vcachemodule.NewVCacheModule()
	err := m.Start("")
	require.NoError(t, err)
	defer m.Stop()
}

// Capture capturing property notifications and retrieving their value
func TestPropertyNotifications(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	const sender1ID = "sender"
	const thing1ID = "thing-1"
	const thing2ID = "thing-2"
	const prop1Name = "prop-1"
	const prop2Name = "prop-2"
	const prop1Value = "value1"

	m := vcachemodule.NewVCacheModule()
	err := m.Start("")
	require.NoError(t, err)
	defer m.Stop()
	m.SetNotificationHook(func(n *msg.NotificationMessage) {
		t.Log("received the notification")
	})

	// Emit notification. A warning is logged as the module cannot forward the notification.
	n1 := msg.NewNotificationMessage(sender1ID, msg.AffordanceTypeProperty, thing1ID, prop1Name, prop1Value)
	m.HandleNotification(n1)
	n2 := msg.NewNotificationMessage(sender1ID, msg.AffordanceTypeProperty, thing2ID, prop1Name, nil)
	m.HandleNotification(n2)
	n3 := msg.NewNotificationMessage(sender1ID, msg.AffordanceTypeProperty, thing2ID, prop2Name, nil)
	m.HandleNotification(n3)

	status := m.GetCacheStatus()
	assert.Equal(t, 2, status.NrThings)

	v, found := m.ReadProperty(thing1ID, prop1Name)
	assert.True(t, found)
	assert.Equal(t, v, prop1Value)

	// A read property request should be answered from the cache
	var respValue any
	req := msg.NewRequestMessage(wot.OpReadProperty, thing1ID, prop1Name, nil, "")
	err = m.HandleRequest(req, func(resp *msg.ResponseMessage) error {
		respValue = resp.Output
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, prop1Value, respValue)
}

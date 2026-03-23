package ncache_test

import (
	"log/slog"
	"os"
	"testing"

	"github.com/hiveot/hivekit/go/modules/vcache"
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

	m := vcache.NewVCacheService()
	err := m.Start("")
	require.NoError(t, err)
	defer m.Stop()
}

// Capture property notifications and retrieving their value
func TestPropertyNotifications(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	const sender1ID = "sender"
	const thing1ID = "thing-1"
	const thing2ID = "thing-2"
	const prop1Name = "prop-1"
	const prop2Name = "prop-2"
	const prop1Value = "value1"
	const prop2Value = "value2"

	m := vcache.NewVCacheService()
	err := m.Start("")
	require.NoError(t, err)
	defer m.Stop()
	m.SetNotificationSink(func(n *msg.NotificationMessage) {
		slog.Info("received the notification")
	})

	// Emit notification. A warning is logged as the module cannot forward the notification.
	n1 := msg.NewNotificationMessage(sender1ID, msg.AffordanceTypeProperty, thing1ID, prop1Name, prop1Value)
	m.HandleNotification(n1)
	n2 := msg.NewNotificationMessage(sender1ID, msg.AffordanceTypeProperty, thing2ID, prop1Name, prop1Value)
	m.HandleNotification(n2)
	n3 := msg.NewNotificationMessage(sender1ID, msg.AffordanceTypeProperty, thing2ID, prop2Name, prop2Value)
	m.HandleNotification(n3)

	status := m.GetCacheStatus()
	assert.Equal(t, 2, status.NrThings)

	// test 1: reading a single property
	notif := m.ReadProperty(thing1ID, prop1Name)
	require.NotNil(t, notif)
	assert.Equal(t, prop1Value, notif.Data)

	// test 2: Thing2 should have multiple values
	prop12Names := []string{prop1Name, prop2Name}
	valueMap, foundAll := m.ReadMultipleProperties(thing2ID, prop12Names)
	require.True(t, foundAll)
	require.Equal(t, 2, len(valueMap))
	assert.Equal(t, prop2Value, valueMap[prop2Name].Data)

	// test 3: a read property request should be answered from the cache
	var respValue string
	req := msg.NewRequestMessage(wot.OpReadProperty, thing1ID, prop1Name, nil, "")
	err = m.HandleRequest(req, func(resp *msg.ResponseMessage) error {
		err = resp.Decode(&respValue)
		return err
	})
	assert.NoError(t, err)
	assert.Equal(t, prop1Value, respValue)

	// test 4: a read multiple properties request should also be answered from the cache
	var multiResp map[string]any
	req = msg.NewRequestMessage(wot.OpReadMultipleProperties, thing2ID, "", prop12Names, "")
	err = m.HandleRequest(req, func(resp *msg.ResponseMessage) error {
		err = resp.Decode(&multiResp)
		return err
	})
	require.NoError(t, err)
	require.Equal(t, 2, len(multiResp))
	require.Equal(t, prop1Value, multiResp[prop1Name])
}

// Capture event notifications and retrieving their value
func TestEventNotifications(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	const sender1ID = "sender"
	const thing1ID = "thing-1"
	const thing2ID = "thing-2"
	const ev1Name = "ev-1"
	const ev2Name = "ev-2"
	const ev1Value = "value1"
	const ev2Value = "value2"

	m := vcache.NewVCacheService()
	err := m.Start("")
	require.NoError(t, err)
	defer m.Stop()
	m.SetNotificationSink(func(n *msg.NotificationMessage) {
		slog.Info("received the notification")
	})

	// Emit notification. A warning is logged as the module cannot forward the notification.
	n1 := msg.NewNotificationMessage(sender1ID, msg.AffordanceTypeEvent, thing1ID, ev1Name, ev1Value)
	m.HandleNotification(n1)
	n2 := msg.NewNotificationMessage(sender1ID, msg.AffordanceTypeEvent, thing2ID, ev1Name, ev1Value)
	m.HandleNotification(n2)
	n3 := msg.NewNotificationMessage(sender1ID, msg.AffordanceTypeEvent, thing2ID, ev2Name, ev2Value)
	m.HandleNotification(n3)

	status := m.GetCacheStatus()
	assert.Equal(t, 2, status.NrThings)

	// test 1: reading a single event notification
	notif := m.ReadEvent(thing1ID, ev1Name)
	require.NotNil(t, notif)
	assert.Equal(t, ev1Value, notif.Data)

	// test 2: RRN read request should return the value
	req := msg.NewRequestMessage(wot.HTOpReadEvent, thing1ID, ev1Name, nil, "")
	err = m.HandleRequest(req, func(resp *msg.ResponseMessage) error {
		var ev msg.NotificationMessage
		err = resp.Decode(&ev)
		require.NoError(t, err)
		assert.Equal(t, ev1Value, ev.Data)
		return nil
	})
	assert.NoError(t, err)
}

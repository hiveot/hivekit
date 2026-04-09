package authz_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	authnapi "github.com/hiveot/hivekit/go/modules/authn/api"
	"github.com/hiveot/hivekit/go/modules/authz"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMain creates a test environment
// Used for all test cases in this package
func TestMain(m *testing.M) {
	utils.SetLogging("info", "")
	res := m.Run()
	if res == 0 {
		// _ = os.RemoveAll(testDir)
	}
	os.Exit(res)
}

// Test starting and stopping authorization service
func TestStartStop(t *testing.T) {
	// cfg := module.NewAuthzConfig()
	svc := authz.NewAuthzService(nil)
	err := svc.Start("")
	require.NoError(t, err)
	svc.Stop()
}

func TestHasPermission(t *testing.T) {
	const clientID1 = "client-1"
	const thingID = "thing1"
	const key = "key1"
	const correlationID = "req-1"
	var testRole = authnapi.ClientRoleViewer

	// handler for providing the role of a client
	getRole := func(clientID string) (role string, err error) {
		if clientID == clientID1 {
			return testRole, nil
		}
		return "", fmt.Errorf("unknown client")
	}
	m := authz.NewAuthzService(getRole)
	err := m.Start("")
	require.NoError(t, err)
	defer m.Stop()

	// check missing clientID
	req := msg.NewRequestMessage(td.OpReadProperty, thingID, key, nil, correlationID)
	req.SenderID = ""
	hasPerm := m.HasPermission(req)
	assert.False(t, hasPerm)

	// check viewers do not have permission to read properties
	req.SenderID = clientID1
	hasPerm = m.HasPermission(req)
	assert.True(t, hasPerm)

	// check viewers do not have permission to publish actions and write-property requests
	req.Operation = td.OpInvokeAction
	req.SenderID = clientID1
	hasPerm = m.HasPermission(req)
	assert.False(t, hasPerm)

	// check operators do have permission to publish actions and write-property requests
	testRole = authnapi.ClientRoleOperator
	hasPerm = m.HasPermission(req)
	assert.True(t, hasPerm)
	testRole = authnapi.ClientRoleManager
	hasPerm = m.HasPermission(req)
	assert.True(t, hasPerm)
	testRole = authnapi.ClientRoleAdmin
	hasPerm = m.HasPermission(req)
	assert.True(t, hasPerm)
	testRole = authnapi.ClientRoleService
	hasPerm = m.HasPermission(req)
	assert.True(t, hasPerm)

	// operators cannot respond with events updates
	//resp := transports.NewResponseMessage(td.OpSubscribeEvent, thingID, key, "eventValue", nil, correlationID)
	//resp.SenderID = operatorID
	//// haspermission only validates requests and event/property notificates are now subscription responses
	//hasPerm = svc.HasPermission(msg.SenderID, msg.Operation, msg.ThingID)
	//assert.False(t, hasPerm)
}

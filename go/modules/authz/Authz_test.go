package authz_test

import (
	"os"
	"testing"

	"github.com/hiveot/hivekit/go/modules/authz/module"
	"github.com/hiveot/hivekit/go/utils"
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
	svc := module.NewAuthzModule(nil)
	err := svc.Start("")
	require.NoError(t, err)
	svc.Stop()
}

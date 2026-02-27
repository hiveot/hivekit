package httpbasicserver_test

import (
	"testing"

	"github.com/hiveot/hivekit/go/modules/transports/httpbasic/httpbasicserver"
	"github.com/hiveot/hivekit/go/modules/transports/tptests"
	"github.com/stretchr/testify/require"
)

// Generic directory store testcases
func TestStartStop(t *testing.T) {
	t.Logf("---%s---\n", t.Name())

	dummyServer := tptests.NewDummyServer("")
	// dummyAuthenticator := authnapi.NewDummyAuthenticator()
	m := httpbasicserver.NewHttpBasicServer(dummyServer)
	err := m.Start("")
	require.NoError(t, err)
	defer m.Stop()
}

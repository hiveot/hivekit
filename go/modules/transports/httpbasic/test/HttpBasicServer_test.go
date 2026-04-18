package httpbasic

import (
	"testing"

	httpbasicpkg "github.com/hiveot/hivekit/go/modules/transports/httpbasic/pkg"
	"github.com/hiveot/hivekit/go/testenv"
	"github.com/stretchr/testify/require"
)

// Generic directory store testcases
func TestStartStop(t *testing.T) {
	t.Logf("---%s---\n", t.Name())

	dummyServer := testenv.NewDummyServer("")
	// dummyAuthenticator := authnapi.NewDummyAuthenticator()
	m := httpbasicpkg.NewHttpBasicServer(dummyServer)
	err := m.Start()
	require.NoError(t, err)
	defer m.Stop()
}

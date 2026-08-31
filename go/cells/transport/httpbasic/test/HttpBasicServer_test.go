package httpbasic_test

import (
	"testing"

	httpbasic_server "github.com/hiveot/hivekit/go/cells/transport/httpbasic/server"
	"github.com/hiveot/hivekit/go/testenv"
	"github.com/stretchr/testify/require"
)

// Generic directory store testcases
func TestStartStop(t *testing.T) {
	t.Logf("---%s---\n", t.Name())

	dummyServer := testenv.NewDummyServer("")
	// dummyAuthenticator := authnapi.NewDummyAuthenticator()
	srv, err := httpbasic_server.StartHttpBasicServer(dummyServer)
	require.NoError(t, err)
	defer srv.Stop()
}

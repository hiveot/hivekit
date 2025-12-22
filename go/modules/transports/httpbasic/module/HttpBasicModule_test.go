package module_test

import (
	"testing"

	"github.com/hiveot/hivekit/go/modules/transports/httpbasic/module"
	"github.com/hiveot/hivekit/go/modules/transports/httpserver"
	"github.com/hiveot/hivekit/go/utils/authn"
	"github.com/stretchr/testify/require"
)

// Generic directory store testcases
func TestStartStop(t *testing.T) {
	t.Logf("---%s---\n", t.Name())

	dummyServer := httpserver.NewDummyServer("")
	dummyAuthenticator := authn.NewDummyAuthenticator()
	m := module.NewHttpBasicModule(dummyServer, dummyAuthenticator)
	err := m.Start()
	require.NoError(t, err)
	defer m.Stop()

}

package module_test

import (
	"testing"

	authnapi "github.com/hiveot/hivekit/go/modules/transports/authn/api"
	"github.com/hiveot/hivekit/go/modules/transports/httpbasic/module"
	"github.com/hiveot/hivekit/go/modules/transports/httptransport"
	"github.com/stretchr/testify/require"
)

// Generic directory store testcases
func TestStartStop(t *testing.T) {
	t.Logf("---%s---\n", t.Name())

	dummyServer := httptransport.NewDummyServer("")
	dummyAuthenticator := authnapi.NewDummyAuthenticator()
	m := module.NewHttpBasicModule(dummyServer, nil, dummyAuthenticator)
	err := m.Start()
	require.NoError(t, err)
	defer m.Stop()
}

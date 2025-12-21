package module_test

import (
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hivekit/go/modules/transports/wothttpbasic/module"
	"github.com/hiveot/hivekit/go/utils/authn"
	"github.com/stretchr/testify/require"
)

// Generic directory store testcases
func TestStartStop(t *testing.T) {
	t.Logf("---%s---\n", t.Name())

	router := chi.NewRouter()
	dummyAuthenticator := authn.NewDummyAuthenticator()
	m := module.NewHttpBasicModule(router, dummyAuthenticator)
	err := m.Start()
	require.NoError(t, err)
	defer m.Stop()

}

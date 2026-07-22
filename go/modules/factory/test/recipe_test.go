package factory_test

import (
	"testing"

	"github.com/hiveot/hivekit/go/api"
	standalonerecipe "github.com/hiveot/hivekit/go/modules/factory/recipes/standalone"
	factory_service "github.com/hiveot/hivekit/go/modules/factory/service"
	"github.com/hiveot/hivekit/go/testenv"
	"github.com/stretchr/testify/require"
)

// 1: setup a test chain

func TestServerRecipe(t *testing.T) {

	env := api.NewAppEnvironment(testDir, false)
	env.HttpsPort = testPort

	// run the module chain for a standalone server
	f := factory_service.NewModuleFactory(env, nil)
	deviceRecipe := standalonerecipe.NewStandAloneDeviceRecipe(f)
	err := deviceRecipe.Start()
	require.NoError(t, err)
	defer deviceRecipe.Stop()

	// run a test device
	testDevice := testenv.NewCounterDevice("", nil)
	err = testDevice.Start()
	require.NoError(t, err)
	defer testDevice.Stop()

}

package factory_test

import (
	"testing"

	"github.com/hiveot/hivekit/go/api"
	factorypkg "github.com/hiveot/hivekit/go/modules/factory/pkg"
	"github.com/hiveot/hivekit/go/modules/factory/recipes"
	"github.com/hiveot/hivekit/go/testenv"
	"github.com/stretchr/testify/require"
)

// 1: setup a test chain

func TestServerRecipe(t *testing.T) {

	env := api.NewAppEnvironment(testDir, false)
	env.HttpsPort = testPort

	// run the module chain for a standalone server
	f := factorypkg.NewModuleFactory(env, nil)
	deviceRecipe := recipes.NewStandAloneDeviceRecipe(f)
	err := deviceRecipe.Start()
	require.NoError(t, err)
	defer deviceRecipe.Stop()

	// run a test device
	testDevice := testenv.NewCounterDevice("", nil)
	err = testDevice.Start()
	require.NoError(t, err)
	defer testDevice.Stop()

}

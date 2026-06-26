package factory_test

import (
	"testing"

	"github.com/hiveot/hivekit/go/modules/factory"
	factorypkg "github.com/hiveot/hivekit/go/modules/factory/pkg"
	"github.com/hiveot/hivekit/go/modules/factory/recipes"
	"github.com/hiveot/hivekit/go/testenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 1: setup a test chain

func TestServerRecipe(t *testing.T) {
	const TestDeviceModuleType = "testDevice"

	env := factory.NewAppEnvironment(testDir, false)
	serverFactory := factorypkg.NewModuleFactory(env, nil)

	// use test device factory
	deviceRecipe := recipes.NewDeviceServerRecipe(serverFactory, &factory.ModuleDefinition{
		Type:        TestDeviceModuleType,
		Constructor: testenv.NewTestDeviceFactory,
	})
	err := deviceRecipe.Start()
	require.NoError(t, err)

	m1 := serverFactory.FindModule(TestDeviceModuleType)
	testDevice, ok := m1.(*testenv.TestDevice)
	_ = testDevice
	assert.True(t, ok)

	deviceRecipe.Stop()
}

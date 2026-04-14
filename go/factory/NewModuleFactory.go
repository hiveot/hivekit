package factory

import (
	factoryapi "github.com/hiveot/hivekit/go/factory/api"
	"github.com/hiveot/hivekit/go/factory/internal"
)

// Create a new module factory
func NewModuleFactory(
	env *factoryapi.AppEnvironment,
	moduleTable map[string]factoryapi.ModuleDefinition) factoryapi.IModuleFactory {

	f := internal.NewModuleFactory(env, moduleTable)
	return f
}

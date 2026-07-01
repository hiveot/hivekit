package vcache

import (
	"github.com/hiveot/hivekit/go/api"
	vcacheapi "github.com/hiveot/hivekit/go/modules/vcache/api"
	"github.com/hiveot/hivekit/go/modules/vcache/internal"
)

// Create a new instance of the value cache server module.
func NewVCacheService() vcacheapi.IVCacheService {
	m := internal.NewVCacheService()
	return m
}

// Create a new instance of the value cache server module using the module factory environment.
func NewVCacheServiceFactory(f api.IModuleFactory, md *api.ModuleDefinition) (api.IHiveModule, error) {
	m := NewVCacheService()
	return m, nil
}

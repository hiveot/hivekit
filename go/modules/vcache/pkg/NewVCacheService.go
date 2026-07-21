package vcachepkg

import (
	"github.com/hiveot/hivekit/go/api"
	vcacheapi "github.com/hiveot/hivekit/go/modules/vcache"
	"github.com/hiveot/hivekit/go/modules/vcache/internal"
)

// Create a new instance of the value cache server module.
func NewValueCacheService() vcacheapi.IValueCacheService {
	m := internal.NewValueCacheService()
	return m
}

// Create a new instance of the value cache server module using the module factory environment.
func NewValueCacheServiceFactory(f api.IModuleFactory, md *api.ModuleDefinition) (api.IHiveModule, error) {
	m := NewValueCacheService()
	return m, nil
}

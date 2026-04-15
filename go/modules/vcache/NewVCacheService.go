package vcache

import (
	factoryapi "github.com/hiveot/hivekit/go/factory/api"
	"github.com/hiveot/hivekit/go/modules"
	vcacheapi "github.com/hiveot/hivekit/go/modules/vcache/api"
	"github.com/hiveot/hivekit/go/modules/vcache/internal"
)

// Create a new instance of the value cache server module.
func NewVCacheService() vcacheapi.IVCacheService {
	m := internal.NewVCacheService()
	return m
}

// Create a new instance of the value cache server module using the module factory environment.
func NewVCacheServiceFactory(f factoryapi.IModuleFactory) modules.IHiveModule {
	m := NewVCacheService()
	return m
}

package vcache

import (
	"github.com/hiveot/hivekit/go/modules"
	factory "github.com/hiveot/hivekit/go/modules/factory"
	vcacheapi "github.com/hiveot/hivekit/go/modules/vcache/api"
	"github.com/hiveot/hivekit/go/modules/vcache/internal"
)

// Create a new instance of the value cache server module.
func NewVCacheService() vcacheapi.IVCacheService {
	m := internal.NewVCacheService()
	return m
}

// Create a new instance of the value cache server module using the module factory environment.
func NewVCacheServiceFactory(f factory.IModuleFactory) modules.IHiveModule {
	m := NewVCacheService()
	return m
}

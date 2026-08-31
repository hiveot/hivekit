package vcache_service

import (
	"github.com/hiveot/hivekit/go/api"
	vcacheapi "github.com/hiveot/hivekit/go/cells/vcache"
	"github.com/hiveot/hivekit/go/cells/vcache/internal"
)

// Create a new instance of the value cache service.
func StartValueCacheService() (vcacheapi.IValueCacheService, error) {
	svc, err := internal.StartValueCacheService()
	return svc, err
}

// Create a new instance of the value cache service using the Cell Factory environment.
func StartValueCacheServiceFactory(f api.ICellFactory, md *api.CellDefinition) (api.IHiveCell, error) {
	svc, err := StartValueCacheService()
	return svc, err
}

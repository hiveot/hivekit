package vcache_service

import (
	"github.com/hiveot/hivekit/go/api"
	vcacheapi "github.com/hiveot/hivekit/go/cells/vcache"
	"github.com/hiveot/hivekit/go/cells/vcache/internal"
)

// Create a ready-to-use value cache service.
func NewValueCacheService() (vcacheapi.IValueCacheService, error) {
	svc, err := internal.NewValueCacheService()
	return svc, err
}

// Create a ready-to-use instance of the value cache service using the Cell Factory environment.
func NewValueCacheServiceFactory(f api.ICellFactory, md *api.CellDefinition) (api.IHiveCell, error) {
	svc, err := NewValueCacheService()
	return svc, err
}

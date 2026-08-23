package vcache_service

import (
	"github.com/hiveot/hivekit/go/api"
	vcacheapi "github.com/hiveot/hivekit/go/cells/vcache"
	"github.com/hiveot/hivekit/go/cells/vcache/internal"
)

// Create a new instance of the value cache service.
func NewValueCacheService() vcacheapi.IValueCacheService {
	m := internal.NewValueCacheService()
	return m
}

// Create a new instance of the value cache service using the Cell Factory environment.
func NewValueCacheServiceFactory(f api.ICellFactory, md *api.CellDefinition) (api.IHiveCell, error) {
	m := NewValueCacheService()
	return m, nil
}

package vcache

import (
	vcacheapi "github.com/hiveot/hivekit/go/modules/vcache/api"
	"github.com/hiveot/hivekit/go/modules/vcache/internal"
)

// Create a new instance of the value cache server module.
func NewVCacheService() vcacheapi.IVCacheService {
	m := internal.NewVCacheService()
	return m
}

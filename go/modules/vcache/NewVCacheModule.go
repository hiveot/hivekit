package vcache

import (
	vcacheapi "github.com/hiveot/hivekit/go/modules/vcache/api"
	VCacheModule "github.com/hiveot/hivekit/go/modules/vcache/internal/module"
)

// Create a new instance of the value cache module.
func NewVCacheModule() vcacheapi.IVCacheModule {
	m := VCacheModule.NewVCacheModule()
	return m
}

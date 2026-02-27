package vcache

import "github.com/hiveot/hivekit/go/modules"

const DefaultVCacheModuleID = "vcache"

// IVCache value-cache module interface.
type IVCacheModule interface {
	modules.IHiveModule
}

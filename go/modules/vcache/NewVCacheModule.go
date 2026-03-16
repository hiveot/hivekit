package vcache

import (
	vcacheapi "github.com/hiveot/hivekit/go/modules/vcache/api"
	vcacheserver "github.com/hiveot/hivekit/go/modules/vcache/internal/server"
)

// Create a new instance of the value cache module.
func NewVCacheServer() vcacheapi.IVCacheServer {
	m := vcacheserver.NewVCacheServer()
	return m
}

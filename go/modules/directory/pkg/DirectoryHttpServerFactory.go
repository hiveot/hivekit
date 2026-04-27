package directorypkg

import (
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/factory"
)

// factory for the directory http interface module
func NewDirectoryServerFactory(f factory.IModuleFactory) modules.IHiveModule {

	httpServer := f.GetHttpServer(true)
	m := NewDirectoryHttpServer(httpServer)
	return m
}

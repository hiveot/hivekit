package directorypkg

import (
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/directory"
	"github.com/hiveot/hivekit/go/modules/factory"
)

const DirectoryModuleType = directory.DirectoryModuleType

// Create the directory service module using the factory environment
func NewDirectoryMsgServerFactory(f factory.IModuleFactory) modules.IHiveModule {
	env := f.GetEnvironment()
	storageDir := env.GetStorageDir(directory.DirectoryModuleType)

	// provide the directory http module instance for inclusing as the TDD base
	httpMod, _ := f.GetModule(directory.DirectoryHttpModuleType, false)
	httpAPI, ok := httpMod.(directory.IDirectoryHttpServer)
	_ = ok
	transports := f.GetTransportServers()

	m := NewDirectoryMsgServer("", storageDir, httpAPI, transports)
	return m
}

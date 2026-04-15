package history

import (
	factoryapi "github.com/hiveot/hivekit/go/factory/api"
	"github.com/hiveot/hivekit/go/modules"
	historyapi "github.com/hiveot/hivekit/go/modules/history/api"
	"github.com/hiveot/hivekit/go/modules/history/internal"
)

// NewHistoryService is the factory method to create a new history service module.
//
// A configuration can be created using: config.NewHistoryConfig(storeDirectory, backend)
func NewHistoryService(config historyapi.HistoryConfig) historyapi.IHistoryService {
	m := internal.NewHistoryService(config)
	return m
}

// Create the history service module using the factory environment
func NewHistoryServiceFactory(f factoryapi.IModuleFactory) modules.IHiveModule {
	env := f.GetEnvironment()
	storageDir := env.GetStorageDir(historyapi.HistoryModuleType)
	config := historyapi.NewHistoryConfig(storageDir, "")
	m := NewHistoryService(config)
	return m
}

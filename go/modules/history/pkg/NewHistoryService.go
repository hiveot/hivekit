package historypkg

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/modules/history"
	"github.com/hiveot/hivekit/go/modules/history/internal"
)

// NewHistoryService is the factory method to create a new history service module.
//
// A configuration can be created using: config.NewHistoryConfig(storeDirectory, backend)
func NewHistoryService(config history.HistoryConfig) history.IHistoryService {
	m := internal.NewHistoryServiceImpl(config)
	return m
}

// Create the history service module using the factory environment
func NewHistoryServiceFactory(f api.IModuleFactory, md *api.ModuleDefinition) (api.IHiveModule, error) {
	env := f.GetEnvironment()
	storageDir := env.GetStorageDir(history.HistoryModuleType)
	config := history.NewHistoryConfig(storageDir, "")
	m := NewHistoryService(config)
	return m, nil
}

package history_service

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/cells/history"
	"github.com/hiveot/hivekit/go/cells/history/internal"
)

// NewHistoryService is the factory method to create a new history service.
//
// A configuration can be created using: config.NewHistoryConfig(storeDirectory, backend)
func NewHistoryService(config history.HistoryConfig) history.IHistoryService {
	m := internal.NewHistoryServiceImpl(config)
	return m
}

// Create the history service using the factory environment
func NewHistoryServiceFactory(f api.ICellFactory, md *api.CellDefinition) (api.IHiveCell, error) {
	env := f.GetEnvironment()
	storageDir := env.GetStorageDir(history.HistoryServiceCellType)
	config := history.NewHistoryConfig(storageDir, "")
	m := NewHistoryService(config)
	return m, nil
}

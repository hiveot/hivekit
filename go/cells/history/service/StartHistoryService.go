package history_service

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/cells/history"
	"github.com/hiveot/hivekit/go/cells/history/internal"
)

// StartHistoryService is the factory method to start a new history service.
//
// A configuration can be created using: config.NewHistoryConfig(storeDirectory, backend)
func StartHistoryService(config history.HistoryConfig) (history.IHistoryService, error) {
	svc, err := internal.StartHistoryServiceImpl(config)
	return svc, err
}

// Start the history service using the factory environment
func StartHistoryServiceFactory(
	f api.ICellFactory, md *api.CellDefinition) (api.IHiveCell, error) {

	env := f.GetEnvironment()
	storageDir := env.GetStorageDir(history.HistoryServiceCellType)
	config := history.NewHistoryConfig(storageDir, "")
	svc, err := StartHistoryService(config)
	return svc, err
}

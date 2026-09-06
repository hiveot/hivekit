package history_service

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/cells/history"
	"github.com/hiveot/hivekit/go/cells/history/internal"
)

// NewHistoryService creates a ready-to-use history tracking service.
// Call Start to publish the TD.
// Notifications and requests passed through this service are stored for later retrieval.
//
// A configuration can be created using: config.NewHistoryConfig(storeDirectory, backend)
func NewHistoryService(config history.HistoryConfig) (history.IHistoryService, error) {
	svc, err := internal.NewHistoryServiceImpl(config)
	return svc, err
}

// NewHistoryServiceFactory creates the history service using the factory environment
func NewHistoryServiceFactory(
	f api.ICellFactory, md *api.CellDefinition) (api.IHiveCell, error) {

	env := f.GetEnvironment()
	storageDir := env.GetStorageDir(history.HistoryServiceCellType)
	config := history.NewHistoryConfig(storageDir, "")
	svc, err := NewHistoryService(config)
	return svc, err
}

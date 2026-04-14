package history

import (
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

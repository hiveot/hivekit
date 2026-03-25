package history

import (
	historyapi "github.com/hiveot/hivekit/go/modules/history/api"
	"github.com/hiveot/hivekit/go/modules/history/internal"
)

// NewHistoryService is the factory method to create a new history service module.
//
//	storeDirectory  is the full path to the directory where to store the history data
//	backend store type as defined in the bucketstore eg, BackendKVBTree or BackendPebble.
func NewHistoryService(storeDirectory string, backend string) historyapi.IHistoryService {
	m := internal.NewHistoryService(storeDirectory, backend)
	return m
}

package historyapi

import (
	"github.com/hiveot/hivekit/go/api/msg"
	bucketstoreapi "github.com/hiveot/hivekit/go/modules/bucketstore/api"
)

// HistoryConfig with history store database configuration
type HistoryConfig struct {
	// Bucket store ID of the backend to store
	// kvbtree, pebble (default), bbolt. See IBucketStore for details.
	Backend string `yaml:"backend"`

	// optional moduleID override for multiple instances
	ModuleID string `yaml:"moduleID"`

	// Bucketstore location where to store the history
	StoreDirectory string `yaml:"storeDirectory"`

	// Default retention from config by event name
	// optional filter for notifications to record. If not provided all notifications are recorded.
	NotificationFilter msg.MessageFilter `yaml:"notifications,omitempty"`
	// optional filter for requests to record. If not provided all requests are recorded.
	RequestFilter msg.MessageFilter `yaml:"requests,omitempty"`
}

// NewHistoryConfig creates a new config with default values
//
// storeDirectory the bucketstore location where to store the history
// backend is the bucketstore backend as described in IBucketStore; "" for default pebble.
func NewHistoryConfig(storeDirectory string, backend string) HistoryConfig {
	if backend == "" {
		backend = bucketstoreapi.BackendPebble
	}
	cfg := HistoryConfig{
		ModuleID:       HistoryModuleType,
		Backend:        backend,
		StoreDirectory: storeDirectory,
	}
	return cfg
}

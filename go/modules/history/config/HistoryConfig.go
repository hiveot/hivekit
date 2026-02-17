package config

import (
	"github.com/hiveot/hivekit/go/modules/bucketstore"
	"github.com/hiveot/hivekit/go/modules/history"
	"github.com/hiveot/hivekit/go/msg"
)

// HistoryConfig with history store database configuration
type HistoryConfig struct {
	// Bucket store ID of the backend to store
	// kvbtree, pebble (default), bbolt. See IBucketStore for details.
	Backend string `yaml:"backend"`

	// optional moduleID override for multiple instances
	ModuleID string `yaml:"moduleID"`

	// Bucket store location where to store the history
	StoreDirectory string `yaml:"storeDirectory"`

	// Default retention from config by event name
	// optional filter for notifications to record. If not provided no notifications are recorded.
	NotificationFilter msg.MessageFilter `yaml:"notifications,omitempty"`
	// optional filter for requests to record. If not provided no requests are recorded.
	RequestFilter msg.MessageFilter `yaml:"requests,omitempty"`
}

// NewHistoryConfig creates a new config with default values
func NewHistoryConfig(storeDirectory string, backend string) HistoryConfig {
	if backend == "" {
		backend = bucketstore.BackendPebble
	}
	cfg := HistoryConfig{
		ModuleID:       history.DefaultHistoryModuleID,
		Backend:        backend,
		StoreDirectory: storeDirectory,
	}
	return cfg
}

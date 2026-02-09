package history

import (
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/msg"
)

// DefaultHistoryModuleID is the default moduleID of the history module.
const DefaultHistoryModuleID = "history"

// DefaultLimit nr items of none provided
const DefaultLimit = 100

// HistoryConfig defines the configuration for the history module.
type HistoryConfig struct {
	// optional filter for notifications to retain. If not provided no notifications are retained.
	NotificationFilter *msg.MessageFilter `yaml:"notifications,omitempty"`
	// optional filter for requests to retain. If not provided no requests are retained.
	RequestFilter *msg.MessageFilter `yaml:"requests,omitempty"`
}

// IHistoryModule defines the interface to the directory service module
// This is implemented in the module and the client api
type IHistoryModule interface {
	modules.IHiveModule
}

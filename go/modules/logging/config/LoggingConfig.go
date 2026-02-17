package config

import (
	"github.com/hiveot/hivekit/go/modules/logging"
	"github.com/hiveot/hivekit/go/msg"
)

// LoggingConfig with logging configuration
type LoggingConfig struct {
	// Logging backend to send the logs to
	// Default is stdout
	Backend string `yaml:"backend"`

	// Logging file or URL where to write the  log
	LogDestination string `yaml:"logDestination"`

	// In addition to the given destination, also log to stdout (default true)
	Log2Stdout bool `yaml:"log2Stdout,omitempty"`

	// Log output as json instead of plain text
	LogAsJson bool `yaml:"logAsJson,omitempty"`

	// logging time format. Defaults to "Jan _2 15:04:05.0000"
	TimeFormat string `yaml:"timeFormat"`

	// optional moduleID override for multiple instances
	ModuleID string `yaml:"moduleID"`

	// optional filter for logging notifications. If not provided all notifications are logged.
	NotificationFilter msg.MessageFilter `yaml:"notifications,omitempty"`

	// optional filter for logging requests. If not provided all requests are logged.
	RequestFilter msg.MessageFilter `yaml:"requests,omitempty"`
}

// NewHistoryConfig creates a new config with default values
//
// logDestination is the default file or URL of the log to write
// backend is the default backend to use
func NewLoggingConfig(logDestination string, backend string) LoggingConfig {
	if backend == "" {
		backend = logging.LoggingBackendFile
	}
	if logDestination == "" {

	}
	cfg := LoggingConfig{
		ModuleID:       logging.DefaultLoggingModuleID,
		Backend:        backend,
		LogDestination: logDestination,
		TimeFormat:     "Jan _2 15:04:05.0000",
		Log2Stdout:     true,
	}
	return cfg
}

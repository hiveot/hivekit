package logging

import (
	loggingapi "github.com/hiveot/hivekit/go/modules/logging/api"
	"github.com/hiveot/hivekit/go/modules/logging/config"
	"github.com/hiveot/hivekit/go/modules/logging/internal"
)

// NewLoggingService creates a new instance of the logging module.
//
// config is the default module configuration.
func NewLoggingService(config config.LoggingConfig) loggingapi.ILoggingService {
	return internal.NewLoggingService(config)
}

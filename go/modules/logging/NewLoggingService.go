package logging

import (
	"path"

	factoryapi "github.com/hiveot/hivekit/go/factory/api"
	"github.com/hiveot/hivekit/go/modules"
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

// NewLoggingServiceFactory creates a new instance of the logging module using the factory environment.
func NewLoggingServiceFactory(f factoryapi.IModuleFactory) modules.IHiveModule {

	// use the application binary as the logfile name
	var logfilename = path.Join(f.GetEnvironment().LogsDir, f.GetEnvironment().AppID)

	config := config.NewLoggingConfig(logfilename, "")
	return internal.NewLoggingService(config)
}

package logging_service

import (
	"path"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/modules/logging"
	"github.com/hiveot/hivekit/go/modules/logging/internal"
)

// NewLoggingService creates a new instance of the logging module.
//
// config is the default module configuration.
func NewLoggingService(config logging.LoggingConfig) logging.ILoggingService {
	return internal.NewLoggingServiceImpl(config)
}

// NewLoggingServiceFactory creates a new instance of the logging module using the factory environment.
func NewLoggingServiceFactory(f api.IModuleFactory, md *api.ModuleDefinition) (api.IHiveModule, error) {

	// use the application binary as the logfile name
	var logfilename = path.Join(f.GetEnvironment().LogsDir, f.GetEnvironment().AppID)

	config := logging.NewLoggingConfig(logfilename, "")
	return NewLoggingService(config), nil
}

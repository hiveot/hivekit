package loggingpkg

import (
	"path"

	"github.com/hiveot/hivekit/go/modules"
	factory "github.com/hiveot/hivekit/go/modules/factory"
	"github.com/hiveot/hivekit/go/modules/logging"
	"github.com/hiveot/hivekit/go/modules/logging/internal"
)

// NewLoggingService creates a new instance of the logging module.
//
// config is the default module configuration.
func NewLoggingService(config logging.LoggingConfig) logging.ILoggingService {
	return internal.NewLoggingService(config)
}

// NewLoggingServiceFactory creates a new instance of the logging module using the factory environment.
func NewLoggingServiceFactory(f factory.IModuleFactory) modules.IHiveModule {

	// use the application binary as the logfile name
	var logfilename = path.Join(f.GetEnvironment().LogsDir, f.GetEnvironment().AppID)

	config := logging.NewLoggingConfig(logfilename, "")
	return internal.NewLoggingService(config)
}

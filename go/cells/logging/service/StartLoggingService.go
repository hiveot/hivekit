package logging_service

import (
	"path"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/cells/logging"
	"github.com/hiveot/hivekit/go/cells/logging/internal"
)

// StartLoggingService creates a new instance of the logging service.
//
// config is the default service configuration.
func StartLoggingService(config logging.LoggingConfig) (logging.ILoggingService, error) {
	return internal.StartLoggingServiceImpl(config)
}

// StartLoggingServiceFactory creates a new instance of the logging service using the factory environment.
func StartLoggingServiceFactory(f api.ICellFactory, md *api.CellDefinition) (api.IHiveCell, error) {

	// use the application binary as the logfile name
	var logfilename = path.Join(f.GetEnvironment().LogsDir, f.GetEnvironment().AppID)

	config := logging.NewLoggingConfig(logfilename, "")
	return StartLoggingService(config)
}

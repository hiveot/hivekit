package logging_service

import (
	"path"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/cells/logging"
	"github.com/hiveot/hivekit/go/cells/logging/internal"
)

// NewLoggingService creates a new instance of the logging service.
//
// config is the default service configuration.
func NewLoggingService(config logging.LoggingConfig) logging.ILoggingService {
	return internal.NewLoggingServiceImpl(config)
}

// NewLoggingServiceFactory creates a new instance of the logging service using the factory environment.
func NewLoggingServiceFactory(f api.ICellFactory, md *api.CellDefinition) (api.IHiveCell, error) {

	// use the application binary as the logfile name
	var logfilename = path.Join(f.GetEnvironment().LogsDir, f.GetEnvironment().AppID)

	config := logging.NewLoggingConfig(logfilename, "")
	return NewLoggingService(config), nil
}

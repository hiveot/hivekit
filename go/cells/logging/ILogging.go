package logging

import "github.com/hiveot/hivekit/go/api"

// Logging destinations
const (
	//  LoggingBackendConsole = "console"
	LoggingBackendFile = "file"

	// TODO: log to syslog
	// LoggingBackendSyslog = "syslog"
)
const LoggingServiceCellType = "logging"

// ILoggingService logging service interface.
type ILoggingService interface {
	api.IHiveCell
}

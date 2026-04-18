package logging

import "github.com/hiveot/hivekit/go/modules"

// Logging destinations
const (
	//  LoggingBackendConsole = "console"
	LoggingBackendFile = "file"

	// TODO: log to syslog
	// LoggingBackendSyslog = "syslog"
)
const LoggingModuleType = "logging"

// ILoggingService logging module interface.
type ILoggingService interface {
	modules.IHiveModule
}

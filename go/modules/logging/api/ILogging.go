package loggingapi

import "github.com/hiveot/hivekit/go/modules"

// Logging destinations
const (
	//  LoggingBackendConsole = "console"
	LoggingBackendFile = "file"

	// TODO: log to syslog
	// LoggingBackendSyslog = "syslog"
)
const DefaultLoggingModuleID = "logging"

// ILoggingService logging module interface.
// This does not have an external API.
type ILoggingService interface {
	modules.IHiveModule
}

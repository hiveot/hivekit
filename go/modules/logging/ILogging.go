package logging

import "github.com/hiveot/hivekit/go/modules"

// Logging destinations
const (
	//  LoggingBackendConsole = "console"
	LoggingBackendFile = "file"

	// TODO: log to syslog
	// LoggingBackendSyslog = "syslog"
)
const DefaultLoggingModuleID = "logging"

// ILoggingModule logging module interface.
// This does not have an external API.
type ILoggingModule interface {
	modules.IHiveModule
}

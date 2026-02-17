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

type ILoggingModule interface {
	modules.IHiveModule
}

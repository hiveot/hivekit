package logging

import "github.com/hiveot/hivekit/go/modules"

const LoggingModuleID = "logging"

type ILoggingModule interface {
	modules.IHiveModule
}

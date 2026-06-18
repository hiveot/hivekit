package reconnect

import (
	"time"

	"github.com/hiveot/hivekit/go/modules"
)

const ReconnectModuleType = "reconnect"

const DefaultMaxReconnectAttempts = 999999
const DefaultBackoffLimit = time.Minute * 5

type IReconnect interface {
	modules.IHiveModule
}

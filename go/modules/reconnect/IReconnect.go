package reconnect

import (
	"time"

	"github.com/hiveot/hivekit/go/api"
)

const ReconnectModuleType = "reconnect"

const DefaultMaxReconnectAttempts = 999999
const DefaultBackoffLimit = time.Minute * 5

type IReconnect interface {
	api.IHiveModule
}

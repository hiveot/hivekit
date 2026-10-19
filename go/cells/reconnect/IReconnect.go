package reconnect

import (
	"time"

	"github.com/hiveot/hivekit/go/api"
)

const ReconnectCellType = "reconnect"
const ReconnectDefaultThingID = "reconnect"

const DefaultMaxReconnectAttempts = 999999
const DefaultBackoffLimit = time.Minute * 3

type IReconnect interface {
	api.IHiveCell

	// Get the connection status of the linked transport client
	GetConnectionStatus() api.ConnectionStatus
}

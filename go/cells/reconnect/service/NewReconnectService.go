package reconnect_service

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/cells/reconnect"
	"github.com/hiveot/hivekit/go/cells/reconnect/internal"
)

// NewReconnectService creates the reconnect service for use with a transport client.
//
// If tpClient is not known at time of creation, then use SetRequestSink to provide
// the client to apply reconnect to.
//
// Call Start to activate the reconnect process. This can call Connect on the client.
//
//	tpClient is the transport client connection instance and sink to use before connecting.
func NewReconnectService(tpClient api.ITransportClient) (reconnect.IReconnect, error) {
	return internal.NewReconnectServiceImpl(tpClient)
}

// Factory for creating a service using the factory environment
func NewReconnectServiceFactory(f api.ICellFactory, md *api.CellDefinition) (api.IHiveCell, error) {
	// env := f.GetEnvironment()

	// option: on start check if the next in the chain is a transport client and register the callback
	return NewReconnectService(nil)
}

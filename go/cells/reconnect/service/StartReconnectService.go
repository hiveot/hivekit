package reconnect_service

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/cells/reconnect"
	"github.com/hiveot/hivekit/go/cells/reconnect/internal"
)

// StartReconnectService starts the reconnect service for use with a transport client.
//
// If cl is not known at time of creation, then SetRequestSink is used to detect
// if the sink is the client to apply reconnect to.
//
// If the client is not connected, Connect will be invoked on the client.
//
//	tpClient is the transport client connection instance and sink to use before connecting.
func StartReconnectService(tpClient api.ITransportClient) (reconnect.IReconnect, error) {
	return internal.StartReconnectServiceImpl(tpClient)
}

// Factory for starting a service using the factory environment
func StartReconnectFactory(f api.ICellFactory, md *api.CellDefinition) (api.IHiveCell, error) {
	// env := f.GetEnvironment()

	// option: on start check if the next in the chain is a transport client and register the callback
	return StartReconnectService(nil)
}

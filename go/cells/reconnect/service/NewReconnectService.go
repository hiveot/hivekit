package reconnect_service

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/cells/reconnect"
	"github.com/hiveot/hivekit/go/cells/reconnect/internal"
)

// NewReconnectService creates a reconnect service for use with a transport client.
//
// If cl is not known at time of creation, then SetRequestSink is used to detect
// if the sink is the client to apply reconnect to.
//
//	sink is the transport client connection instance and sink to use before connecting.
func NewReconnectService(sink api.ITransportClient) reconnect.IReconnect {
	svc := internal.NewReconnectServiceImpl(sink)

	return svc
}

// Factory for creating a service using the factory environment
func NewReconnectFactory(f api.ICellFactory, md *api.CellDefinition) (api.IHiveCell, error) {
	// env := f.GetEnvironment()

	// option: on start check if the next in the chain is a transport client and register the callback
	svc := NewReconnectService(nil)
	return svc, nil
}

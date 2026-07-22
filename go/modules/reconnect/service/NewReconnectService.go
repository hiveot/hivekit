package reconnect_service

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/modules/reconnect"
	"github.com/hiveot/hivekit/go/modules/reconnect/internal"
)

// NewReconnectService creates a reconnect module for use with a transport client.
//
// If cl is not known at time of creation, then SetRequestSink is used to detect
// if the sink is the client to apply reconnect to.
//
//	sink is the transport client connection instance and sink to use before connecting.
func NewReconnectService(sink api.ITransportClient) reconnect.IReconnect {
	m := internal.NewReconnectServiceImpl(sink)

	return m
}

// Factory for creating a module using the factory environment
func NewReconnectFactory(f api.IModuleFactory, md *api.ModuleDefinition) (api.IHiveModule, error) {
	// env := f.GetEnvironment()

	// option: on start check if the next in the chain is a transport client and register the callback
	c := NewReconnectService(nil)
	return c, nil
}

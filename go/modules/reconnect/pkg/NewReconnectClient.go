package reconnectpkg

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/modules/reconnect"
	"github.com/hiveot/hivekit/go/modules/reconnect/internal"
)

// NewReconnectClient creates a reconnect module for use with the given client.
//
// If cl is not known at time of creation, then SetRequestSink is used to detect
// if the sink is the client to apply reconnect to.
//
//	sink is the transport client connection instance and sink to use before connecting.
func NewReconnectClient(sink api.ITransportClient) reconnect.IReconnect {
	m := internal.NewReconnectClientImpl(sink)

	return m
}

// Factory for creating a consumer module using the factory environment
func NewReconnectFactory(f api.IModuleFactory, md *api.ModuleDefinition) (api.IHiveModule, error) {
	// env := f.GetEnvironment()

	// option: on start check if the next in the chain is a transport client and register the callback
	c := NewReconnectClient(nil)
	return c, nil
}

package reconnectpkg

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/modules/reconnect"
	"github.com/hiveot/hivekit/go/modules/reconnect/internal"
)

// NewReconnectClient creates a reconnect module for use with the given client.
//
//	cl is the transport client connection instance to use before connecting
func NewReconnectClient(cl api.ITransportClient) reconnect.IReconnect {
	m := internal.NewReconnectClientImpl(cl)

	return m
}

// Factory for creating a consumer module using the factory environment
func NewReconnectFactory(f api.IModuleFactory, md *api.ModuleDefinition) (api.IHiveModule, error) {
	// env := f.GetEnvironment()
	// TODO: figure out how to include this in a recipe without knowing what client to use
	// option: on start check if the next in the chain is a transport client and register the callback
	c := NewReconnectClient(nil)
	return c, nil
}

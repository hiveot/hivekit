package router

import (
	routerapi "github.com/hiveot/hivekit/go/modules/router/api"
	"github.com/hiveot/hivekit/go/modules/router/internal/module"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/wot/td"
)

// Create a new instance of the router module with the default module ID.
// Start must be called before usage.
func NewRouterModule(getTD func(thingID string) *td.TD, transports []transports.ITransportServer) routerapi.IRouterModule {
	m := module.NewRouterModule(getTD, transports)
	return m
}

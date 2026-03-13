package router

import (
	routerapi "github.com/hiveot/hivekit/go/modules/router/api"
	"github.com/hiveot/hivekit/go/modules/router/internal/module"
)

func NewRouterModule() routerapi.IRouterModule {
	m := module.NewRouterModule()
	return m
}

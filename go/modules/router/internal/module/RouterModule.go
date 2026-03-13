package module

import (
	"github.com/hiveot/hivekit/go/modules"
	routerapi "github.com/hiveot/hivekit/go/modules/router/api"
)

type RouterModule struct {
	modules.HiveModuleBase
}

func (m *RouterModule) Start(_ string) (err error) {
	return err
}

func (m *RouterModule) Stop() {
}

func NewRouterModule() *RouterModule {
	m := &RouterModule{}
	m.SetModuleID(routerapi.DefaultRouterServiceID)

	return m
}

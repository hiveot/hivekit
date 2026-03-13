package routerapi

import "github.com/hiveot/hivekit/go/modules"

const DefaultRouterServiceID = "router"

type IRouterModule interface {
	modules.IHiveModule
}

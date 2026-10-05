package rcrouter

import "github.com/hiveot/hivekit/go/api"

const RCRouterCellType = "rcrouter"

const RCRouterDefaultThingID = "rcrouter"

// The rcrouter service routes request to reverse-connected devices.
type IRCRouterService interface {
	api.IHiveCell
}

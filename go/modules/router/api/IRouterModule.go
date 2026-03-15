package routerapi

import "github.com/hiveot/hivekit/go/modules"

const DefaultRouterServiceID = "router"

type IRouterModule interface {
	modules.IHiveModule

	// Determine if the thing is reachable by the router.
	//
	// This returns true if a client connection is established by the router, or if
	// a reverse connection exists by the thing agent.
	IsReachable(thingID string) bool

	// Return the ISO timestamp when the Thing was last seen by the router.
	// This returns an empty string if no known record exists.
	LastSeen(thingID string) string
}

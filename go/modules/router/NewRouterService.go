package router

import (
	"crypto/x509"

	routerapi "github.com/hiveot/hivekit/go/modules/router/api"
	"github.com/hiveot/hivekit/go/modules/router/internal"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/wot/td"
)

// NewRouterService creates a new instance of the router service module with the default module ID.
// Start must be called before usage.
//
//	storageDir location where the module stores its data
//	getTD is the handler to lookup a TD for a thingID from a directory
//	transports is a list of transport servers that can contain reverse agent connections.
//	caCert is the CA certificate used to verify device connections
func NewRouterService(
	storageDir string,
	getTD func(thingID string) *td.TD,
	transports []transports.ITransportServer,
	caCert *x509.Certificate,
) routerapi.IRouterService {

	m := internal.NewRouterService(storageDir, getTD, transports, caCert)
	return m
}

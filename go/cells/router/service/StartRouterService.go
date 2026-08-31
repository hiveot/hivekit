package router_service

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"time"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/cells/directory"
	"github.com/hiveot/hivekit/go/cells/router"
	"github.com/hiveot/hivekit/go/cells/router/internal"
)

// When factory instantiated the router can enable auto-reconnect for new connections.
// See also SetAutoReconnect()
// TODO: use config to set auto-reconnect. For now don't because it might hide auth problems.
const DefaultRouterAutoConnect = false

// StartRouterService creates a new instance of the router service with the default router typeID.
// Start must be called before usage.
//
//	storageDir location where the router stores its data
//	autoReconnect to enable auto-reconnecting of dropped client connections, restoring subscriptions.
//	clientID default clientID to connect if no other credentials are known
//	clientCert optional client certificate to use for mutual authentication - overrides clientID
//	rootCAs are the CA certificates used to verify device connections
//	timeout is the maximum wait time for sending requests to clients.
//	getTD  handler to lookup a TD for a thingID from a directory
//	getSrv handler to return the running list of transport servers that can contain
//	 reverse connections. nil to not support RCs.
func StartRouterService(storageDir string,
	autoReconnect bool,
	clientID string,
	clientCert *tls.Certificate,
	rootCAs *x509.CertPool, timeout time.Duration,
	getTD func(thingID string) *td.TD,
	getSrv func() []api.ITransportServer,
) (router.IRouterService, error) {

	return internal.StartRouterServiceImpl(storageDir, autoReconnect,
		clientID, clientCert, rootCAs, timeout, getTD, getSrv)
}

// Create a router service instance using the factory environment.
// This needs a directory client or service with a getTD method to lookup a Thing TD.
func StartRouterServiceFactory(f api.ICellFactory, md *api.CellDefinition) (api.IHiveCell, error) {

	var getTD func(string) *td.TD
	env := f.GetEnvironment()
	storageDir := env.GetStorageDir(router.RouterCellType)

	// The router can be used with a directory server or client. Try both.
	m, err := f.StartCell(directory.DirectoryServiceCellType, true)
	if err == nil {
		if dirMod, ok := m.(directory.IDirectoryService); ok {
			getTD = dirMod.GetTD
		}
	} else {
		// maybe directory client?
		m, err = f.StartCell(directory.DirectoryClientCellType, true)
		if err == nil {
			if dirMod, ok := m.(directory.IDirectoryClient); ok {
				getTD = dirMod.Cache().GetThing
			}
		}
	}
	if err != nil {
		return nil, fmt.Errorf("NewRouterServiceFactory. Missing directory client or service.")
	}
	// TODO: use config to set auto-reconnect. For now don't because it might hide auth problems.
	autoReconnect := DefaultRouterAutoConnect
	timeout := f.GetEnvironment().RpcTimeout
	clientCert, _ := env.GetClientCert()
	svc, err := StartRouterService(
		storageDir, autoReconnect, env.ClientID, clientCert, env.GetRootCAs(),
		timeout, getTD, f.GetTransportServers)
	svc.SetTimeout(env.RpcTimeout)

	return svc, err
}

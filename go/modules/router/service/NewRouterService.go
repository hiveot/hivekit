package router_service

import (
	"crypto/x509"
	"fmt"
	"time"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules/directory"
	"github.com/hiveot/hivekit/go/modules/router"
	"github.com/hiveot/hivekit/go/modules/router/internal"
)

// NewRouterService creates a new instance of the router service module with the default module ID.
// Start must be called before usage.
//
//	storageDir location where the module stores its data
//	autoReconnect to enable auto-reconnecting of dropped client connections, restoring subscriptions.
//	getTD  handler to lookup a TD for a thingID from a directory
//	getSrv handler to return the running list of transport servers that can contain
//	 reverse connections. nil to not support RCs.
//	rootCAs are the CA certificates used to verify device connections
//	timeout is the maximum wait time for sending requests to clients.
func NewRouterService(storageDir string,
	autoReconnect bool,
	getTD func(thingID string) *td.TD,
	getSrv func() []api.ITransportServer,
	rootCAs *x509.CertPool, timeout time.Duration,
) router.IRouterService {

	m := internal.NewRouterServiceImpl(storageDir, autoReconnect, getTD, getSrv, rootCAs, timeout)
	return m
}

// Create a router service instance using the factory environment
// This loads the directory module to lookup a Thing TD
func NewRouterServiceFactory(f api.IModuleFactory, md *api.ModuleDefinition) (api.IHiveModule, error) {
	var getTD func(string) *td.TD
	env := f.GetEnvironment()
	storageDir := env.GetStorageDir(router.RouterModuleType)

	// The router can be used server or client side. Check for both server and client directory.
	m, err := f.StartModule(directory.DirectoryServiceModuleType, true)
	if err == nil {
		if dirMod, ok := m.(directory.IDirectoryService); ok {
			getTD = dirMod.GetTD
		}
	} else {
		// maybe directory client?
		m, err = f.StartModule(directory.DirectoryClientModuleType, true)
		if err == nil {
			if dirMod, ok := m.(directory.IDirectoryClient); ok {
				getTD = dirMod.Cache().GetThing
			}
		}
	}
	if err != nil {
		return nil, fmt.Errorf("NewRouterServiceFactory. Missing TD directory: %w", err)
	}
	// TODO: use config to set auto-reconnect. For now don't because it might hide auth problems.
	autoReconnect := false
	timeout := f.GetEnvironment().RpcTimeout
	svc := NewRouterService(
		storageDir, autoReconnect, getTD, f.GetTransportServers, env.GetRootCAs(), timeout)
	svc.SetTimeout(env.RpcTimeout)

	return svc, nil
}

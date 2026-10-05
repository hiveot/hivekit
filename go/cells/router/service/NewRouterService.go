package router_service

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/cells/directory"
	"github.com/hiveot/hivekit/go/cells/router"
	"github.com/hiveot/hivekit/go/cells/router/service/internal"
)

// When factory instantiated the router can enable auto-reconnect for new connections.
// See also SetAutoReconnect()
// TODO: use config to set auto-reconnect. For now don't because it might hide auth problems.
const DefaultRouterAutoConnect = false

// NewRouterService creates a ready-to-use instance of the router service with the default router typeID.
// Start must be called before usage.
//
//	storageDir for the credentials storage directory, "" for in-memory testing
//	autoReconnect to enable auto-reconnecting of dropped client connections, restoring subscriptions.
//	clientID default clientID to connect if no other credentials are known
//	clientCert optional client certificate to use for mutual authentication - overrides clientID
//	rootCAs are the CA certificates used to verify device connections
//	getTD  handler to lookup a TD for a thingID from a directory. Required.
func NewRouterService(storageDir string,
	autoReconnect bool,
	clientID string,
	clientCert *tls.Certificate,
	rootCAs *x509.CertPool,
	getTD func(thingID string) *td.TD,
) (router.IRouterService, error) {

	return internal.NewRouterServiceImpl(storageDir, autoReconnect,
		clientID, clientCert, rootCAs, getTD)
}

// Create a router service instance using the factory environment.
//
// If the factory environment contains a directory client or server, use its getTD
// provider of TD's.
//
// If the factory environment contains a client certificate then include it for
// authentication to devices and services.
//
// This needs a directory client or service with a getTD method to lookup a Thing TD.
func NewRouterServiceFactory(f api.ICellFactory, md *api.CellDefinition) (api.IHiveCell, error) {

	var getTD func(string) *td.TD
	env := f.GetEnvironment()
	storageDir := env.GetStorageDir(router.RouterCellType)

	// The router can be used with a directory server or client. Try both.
	m, err := f.NewCell(directory.DirectoryServiceCellType, true)
	if err == nil {
		if dirMod, ok := m.(directory.IDirectoryService); ok {
			getTD = dirMod.GetTD
		}
	} else {
		// maybe directory client?
		m, err = f.NewCell(directory.DirectoryClientCellType, true)
		if err == nil {
			if dirMod, ok := m.(directory.IDirectoryClient); ok {
				getTD = dirMod.GetTD
			}
		}
	}
	if err != nil {
		return nil, fmt.Errorf("NewRouterServiceFactory. Missing directory client or service.")
	}

	// configure the direct gateway endpoint
	if env.GatewayURL != "" {
		// default route is the gateway instead of creating client connections

		// if the router is used in a gateway then this forwards requests to another gateway
		// TODO: check on start if this gateway is not the router's server itself
		//  that would cause an endless loop.
		// TODO: should a gateway client be used in a separate recipe
		//  or should consumers be able to use both a router and gateway in parallel?
	}

	// TODO: use config to set auto-reconnect. For now don't because it might hide auth problems.
	autoReconnect := DefaultRouterAutoConnect
	clientCert, _ := env.GetClientCert()
	svc, err := NewRouterService(
		storageDir, autoReconnect,
		env.ClientID, clientCert,
		env.GetRootCAs(),
		getTD,
	)

	return svc, err
}

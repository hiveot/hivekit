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
// If env contains a gateway URL without TD then getTD returns a gateway TD.
// If env contains a predefined serverTD then use it for all connections.
// If the factory environment contains a directory client or server, use its getTD
// provider of TD's.
//
// If the factory environment contains a client certificate then include it for
// authentication to devices and services.
//
// This needs a directory client or service with a getTD method to lookup a Thing TD.
func NewRouterServiceFactory(f api.ICellFactory, md *api.CellDefinition) (api.IHiveCell, error) {

	var getTD func(string) *td.TD
	var err error
	var serverID string
	env := f.GetEnvironment()
	storageDir := env.GetStorageDir(router.RouterCellType)

	// if a gateway URL is preconfigured then getTD returns this TD
	// TODO: should this be a environment configuration?
	if env.GatewayURL != "" {
		serverID := "gateway"
		gwTD := td.NewTD(serverID, "dummy gateway", "Gateway")
		gwTD.Base = env.GatewayURL
		//
		getTD = func(_ string) *td.TD {
			return gwTD
		}
		// FIXME: provide auth token/cert
	} else {
		// if a server TD is preconfigured then getTD returns this TD
		serverTD := env.GetServerTD()
		if serverTD != nil {
			serverID = serverTD.ID
			getTD = func(_ string) *td.TD {
				return env.GetServerTD()
			}
		}
	}
	if getTD == nil {
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
	}
	if err != nil {
		return nil, fmt.Errorf("NewRouterServiceFactory. Missing directory client or service.")
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
	// set configured credentials, if any
	authToken, _ := env.GetAuthToken()
	if authToken != "" {
		svc.AddCredentials(serverID, env.ClientID, authToken, td.SecSchemeBearer)
	}
	return svc, err
}

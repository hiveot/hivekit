package router_service

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/wot"
	"github.com/hiveot/hivekit/go/cells/directory"
	"github.com/hiveot/hivekit/go/cells/router"
	"github.com/hiveot/hivekit/go/cells/router/service/internal"
)

// When factory instantiated the router can enable auto-reconnect for new connections.
// See also SetAutoReconnect()
// TODO: use config to set auto-reconnect.
const DefaultRouterAutoConnect = true

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
	getTD func(thingID string) *wot.TD,
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

	var getTD func(string) *wot.TD
	var err error
	var serverID string
	env := f.GetEnvironment()
	storageDir := env.GetStorageDir(router.RouterCellType)

	// if a serverURL override is set then force the router to always connect to it.
	if env.ServerURL != "" {
		serverID := "gateway"
		gwTD := wot.NewTD(serverID, "dummy gateway", "Gateway")
		gwTD.Base = env.ServerURL
		// 	//
		getTD = func(_ string) *wot.TD {
			return gwTD
		}
		// } else {
		// if a server TD is preconfigured then getTD returns this TD
		// serverTD := env.GetServerTD()
		// if serverTD != nil {
		// serverID = serverTD.ID
		// getTD = func(_ string) *wot.TD {
		// return env.GetServerTD()
		// }
		// }
	}
	if getTD == nil {
		var cell api.IHiveCell
		// The router can be used with a directory server or client. Try both.
		cell, err = f.NewCell(directory.DirectoryServiceCellType, true)
		if err == nil {
			if dirMod, ok := cell.(directory.IDirectoryService); ok {
				getTD = dirMod.GetTD
			}
		} else {
			// maybe directory client?
			cell, err = f.NewCell(directory.DirectoryClientCellType, true)
			if err == nil {
				if dirMod, ok := cell.(directory.IDirectoryClient); ok {
					getTD = dirMod.GetTD
				}
			}
		}
	}
	if err != nil {
		return nil, fmt.Errorf("NewRouterServiceFactory. Missing directory client or service.")
	}

	autoReconnect := DefaultRouterAutoConnect
	clientCert, _ := env.GetClientCert(env.ClientID)
	svc, err := NewRouterService(
		storageDir, autoReconnect,
		env.ClientID, clientCert,
		env.GetRootCAs(),
		getTD,
	)
	// set configured credentials, if any
	authToken, _ := env.GetAuthToken()
	if authToken != "" {
		svc.AddCredentials(serverID, env.ClientID, authToken, wot.SecSchemeBearer)
	}
	return svc, err
}

package gateway_recipe

import (
	"crypto/tls"
	"os"
	"time"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/api/vocab"
	"github.com/hiveot/hivekit/go/cells"
	"github.com/hiveot/hivekit/go/cells/authn"
	authn_service "github.com/hiveot/hivekit/go/cells/authn/service"
	"github.com/hiveot/hivekit/go/cells/authz"
	authz_service "github.com/hiveot/hivekit/go/cells/authz/service"
	"github.com/hiveot/hivekit/go/cells/certs"
	certs_service "github.com/hiveot/hivekit/go/cells/certs/service"
	"github.com/hiveot/hivekit/go/cells/directory"
	directory_service "github.com/hiveot/hivekit/go/cells/directory/service"
	"github.com/hiveot/hivekit/go/cells/history"
	history_service "github.com/hiveot/hivekit/go/cells/history/service"
	"github.com/hiveot/hivekit/go/cells/logging"
	logging_service "github.com/hiveot/hivekit/go/cells/logging/service"
	"github.com/hiveot/hivekit/go/cells/rcrouter"
	rcrouter_service "github.com/hiveot/hivekit/go/cells/rcrouter/service"
	"github.com/hiveot/hivekit/go/cells/router"
	router_service "github.com/hiveot/hivekit/go/cells/router/service"
	"github.com/hiveot/hivekit/go/cells/transport/discovery"
	discovery_server "github.com/hiveot/hivekit/go/cells/transport/discovery/server"
	grpc "github.com/hiveot/hivekit/go/cells/transport/grpc"
	grpc_server "github.com/hiveot/hivekit/go/cells/transport/grpc/server"
	tls_server "github.com/hiveot/hivekit/go/cells/transport/tlsserver/server"
	"github.com/hiveot/hivekit/go/cells/transport/wss"
	wss_server "github.com/hiveot/hivekit/go/cells/transport/wss/server"
	factory_service "github.com/hiveot/hivekit/go/factory/service"
)

// GatewayRecipeCells defines a cell chain of an application gateway.
//
// # IN DEVELOPMENT - NOT READY YET
//
// The application gateway provides protocol servers, authentication, a directory,
// a router for communication with connected devices, and more.
var GatewayRecipeCells = []api.CellDefinition{
	{
		// If no CA certificate is found in the AppEnvironment then generate a CA.
		// If no server certificate is found in the AppEnvironment then generate a self-signed certificate.
		Type:        certs.InitFactoryCertsCellType,
		Constructor: certs_service.RunInitFactoryCerts,
	},
	{
		// http server is needed by websocket transport server
		// It uses the factory registered authenticator.
		Type:        api.HttpServerCellType,
		Constructor: tls_server.NewTLSServerFactory,
	},
	// --- nested recipe with the servers operating in parallel
	{
		// requests are passed to all servers until one accepts
		Type:        api.BusRecipeType,
		Constructor: factory_service.StartBusFormationFactory,
		Config: []api.CellDefinition{
			// {
			// 	// http-basic transport server
			// 	Type:        httpbasic.HttpBasicServerCellType,
			// 	Constructor: httpbasic_server.NewHttpBasicServerFactory,
			// },
			{
				// Websocket transport server
				Type:        wss.WotWebsocketServerCellType,
				Constructor: wss_server.NewWotWssServerFactory,
			},
			// {
			// 	// Hiveot SSE
			// 	Type:        ssesc.SseScServerCellType,
			// 	Constructor: ssesc_server.StartSseScServerFactory,
			// },
			{
				// Hiveot gRPC
				Type:        grpc.HiveotGrpcServerCellType,
				Constructor: grpc_server.NewHiveotGrpcServerFactory,
			},
			// {
			// 	// MQTT server
			// 	Type:        mqtt.MqttServerCellType,
			// 	Constructor: mqttpkg.NewMqttServerFactory,
			// },
			// {
			// 	// MQTT client
			// 	Type:        mqttgw.MqttClientCellType,
			// 	Constructor: mqttgwpkg.NewMqttClientFactory,
			// },
		},
	},
	{
		// logging of requests
		Type:        logging.LoggingServiceCellType,
		Constructor: logging_service.NewLoggingServiceFactory,
	},
	{
		// Authorization of remote requests
		Type:        authz.AuthzServiceCellType,
		Constructor: authz_service.NewAuthzServiceFactory,
	},
	{
		// Authentication handler and service
		Type:        authn.AuthnServiceCellType,
		Constructor: authn_service.NewAuthnServiceFactory,
	},

	{
		// Certificate management
		Type:        certs.CertsServiceCellType,
		Constructor: certs_service.NewCertsServiceFactory,
	},

	{
		// request and notification history storage
		Type:        history.HistoryServiceCellType,
		Constructor: history_service.NewHistoryServiceFactory,
	},

	// not in a gateway. The gateway stores the TD's as-is and uses them to forward
	// requests locally.
	// all TDs added to the gateway will have their address/security/base updated
	// to the gateway itself.
	// {
	// 	// add forms to update the published TD with appropriate forms
	// 	Type:        addforms.AddFormsCellType,
	// 	Constructor: addforms_service.NewAddFormsServiceFactory,
	// },

	{
		// Directory service
		Type:        directory.DirectoryServiceCellType,
		Constructor: directory_service.NewDirectoryServiceFactory,
	},
	{
		// discovery of the directory (must be placed after directory)
		Type:        discovery.DiscoveryServerCellType,
		Constructor: discovery_server.NewDiscoveryServerFactory,
	},

	{
		// Router service for routing requests to devices
		// this requires a directory client or service.
		Type:        router.RouterCellType,
		Constructor: router_service.NewRouterServiceFactory,
	},
	{
		// RC service for routing requests to reverse connections.
		Type:        rcrouter.RCRouterCellType,
		Constructor: rcrouter_service.NewRCRouterServiceFactory,
	},

	// todo: optional logging of requests
	// todo: optional authorization of requests
}

// This gateway recipe creates a server that forwards requests to
// devices. The gateway is a single point of access with authentication,
// authorization and logging. Device credentials are managed by the
// gateway router so consumers don't use direct access to devices.
//
// Since consumers don't need access to individual devices they
// ignore the forms in the TD. They only need a gateway connection
// and send requests using one of the supported protocols.
//
// The gateway publishes its own discovery record of type Gateway.
//
// This recipe can be modified to use an external directory by replacing
// the directory service cell with a directory client cell and configuring
// it with the remote directory TD. Both support the same API.
type GatewayRecipe struct {
	*cells.HiveCellBase
	f         api.ICellFactory
	formation api.IRecipe
}

// Add an account for connecting to the gateway.
// This is a convenience function that locates the authn and certs services and
// creates an account, optional token and optional key and TLS certificate.
//
//	clientID of the account to add
//	name is the display name of the account
//	role is the account role, ClientRoleViewer, ClientRoleDevice, ...
//	withToken generates a new token file named {clientID}.token
//	withCert generates a new client certificate named {clientID}Cert/Key.pem
//
// This returns the token and client cert if requested
func (r *GatewayRecipe) AddAccount(
	clientID string, name string, role string, withToken bool, withCert bool,
) (token string, clientCert *tls.Certificate, err error) {

	var tlsCert *tls.Certificate

	authnSvc := r.f.GetCell(authn.AuthnServiceCellType).(authn.IAuthnService)
	// ensure the client exists
	_ = authnSvc.AddClient(clientID, name, role)
	if withToken {
		// use the default validation period
		token, _, err = authnSvc.GetSessionManager().CreateToken(clientID, 0)
		err = authnSvc.GetSessionManager().SaveToken(clientID, token)
	}
	if withCert {
		certSvc := r.f.GetCell(certs.CertsServiceCellType).(certs.ICertsService)
		validity := time.Hour * 24 * 365
		tlsCert, err = certSvc.CreateClientTLSCert(clientID, role, validity)
		if err != nil {
			return "", nil, err
		}
		_ = tlsCert
	}
	return token, clientCert, err
}

// Return the certificate manager in this recipe
func (r *GatewayRecipe) GetCertsSvc() certs.ICertsService {
	certsSvc := api.GetFactoryCell[certs.ICertsService](r.f, certs.CertsServiceCellType)
	return certsSvc
}

// Start the gateway recipe and factory.
// This serves the gateway TD discovery
func (r *GatewayRecipe) Start() {
	// call the cells
	r.f.Start()

	// TODO: add gateway as device with props and events
	tdoc := td.NewTD(r.GetThingID(), "HiveOT Gateway", vocab.DeviceNetGateway)
	r.f.AddTDSecForms(tdoc, false)

	discoSrv := api.GetFactoryCell[discovery.IDiscoveryServer](
		r.f, discovery.DiscoveryServerCellType)
	instanceName := r.GetThingID()
	discoSrv.ServeGatewayTD(instanceName, tdoc)
}

// Stop the recipe and factory.
func (r *GatewayRecipe) Stop() {
	r.f.Stop()
}

// NewGatewayRecipe creates an IoT gateway.
//
// Invoke Start to Start the cells in the recipe.
//
// Intended as the central connection point for consumers, services, RC devices,
// and external devices whose TD exists in the directory.
//
// This:
// 1. manages certificates
// 2. manages users and handles authentication
// 3. provides a directory service
// 4. supports directory discovery
// 5. runs protocol servers for http-basic, websockets, grpc and others
// 6. Option to include a digital twin service
//
// Cell chain:
//
//	 -> init certs
//		  -> server group [http, wss, sse, mqtt]
//		     -> logging
//	           -> authz
//		          -> authn
//		             -> history
//		                -> directory
//		                   -> discovery server
//	                         -> router | reconnect | clients
//
// This returns the recipe which can be used like any other cell, along with the
// factory used to create the recipe or an error.
// The factory can be used to lookup cells.
// Call Start on the recipe to run the gateway.
func NewGatewayRecipe(env *api.HiveEnvironment) (
	r *GatewayRecipe, f api.ICellFactory, err error) {

	f = factory_service.NewCellFactory(env, nil)

	formation, err := factory_service.NewChainFormation(f, GatewayRecipeCells, nil)
	if err != nil {
		return nil, nil, err
	}

	hostname, _ := os.Hostname()
	thingID := hostname + ":gateway"
	r = &GatewayRecipe{
		HiveCellBase: cells.NewHiveCellBase(thingID),
		f:            f,
		formation:    formation,
	}
	r.SetTimeout(env.RpcTimeout)
	var _ api.IRecipe = r // interface checks
	return r, f, err
}

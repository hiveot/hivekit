package testenv

import (
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules/agent"
	"github.com/hiveot/hivekit/go/modules/transport"
	grpcpkg "github.com/hiveot/hivekit/go/modules/transport/grpc/pkg"
	httpbasicpkg "github.com/hiveot/hivekit/go/modules/transport/httpbasic/pkg"
	"github.com/hiveot/hivekit/go/modules/transport/httptransport"
	httptransportpkg "github.com/hiveot/hivekit/go/modules/transport/httptransport/pkg"
	ssescpkg "github.com/hiveot/hivekit/go/modules/transport/ssesc/pkg"
	wsspkg "github.com/hiveot/hivekit/go/modules/transport/wss/pkg"
)

// TestDevice contains a server and agent for testing and simulation
type TestDevice struct {
	agentID         string
	authenticator   transport.IAuthenticator
	cfg             *httptransport.Config
	HttpServer      transport.IHttpServer
	TransportServer transport.ITransportServer
	Agent           *agent.Agent
	// the server protocol to use, eg ProtocolTypeWotWSS, ...
	protocolType string

	td *td.TD
}

// return the TD of the test device
func (device *TestDevice) GetTD() *td.TD {
	return device.td
}

// set the request sink of the test device
func (device *TestDevice) SetRequestSink(handler msg.RequestHandler) {
	device.Agent.SetRequestSink(handler)
}

// Start the test device
// This:
// 1. starts the http server
// 2. creates the protocol transport server
// 3. add transport forms to the test device
// 4. create and link an agent that handles requests
func (device *TestDevice) Start() error {

	// setup the server, transport and link the device to the transport
	// cfg := httpserverapi.NewConfig(addr, port, serverCert, caCert, validateToken)
	device.HttpServer = httptransportpkg.NewHttpTransportServer(
		device.cfg, device.authenticator)

	err := device.HttpServer.Start()
	if err != nil {
		device.HttpServer = nil
		return err
	}
	switch device.protocolType {
	case transport.ProtocolTypeWotHttpBasic:
		device.TransportServer = httpbasicpkg.NewHttpBasicServer(device.HttpServer)
	case transport.ProtocolTypeHiveotGrpc:
		device.TransportServer = grpcpkg.NewHiveotGrpcServer(
			"", device.cfg.ServerCert, device.cfg.CaCert, device.authenticator, 0)
	case transport.ProtocolTypeHiveotSsesc:
		device.TransportServer = ssescpkg.NewSseScServer(device.HttpServer, 0)
	case transport.ProtocolTypeWotWebsocket:
		device.TransportServer = wsspkg.NewWotWssServer(device.HttpServer, 0)
	case transport.ProtocolTypeHiveotWebsocket:
		device.TransportServer = wsspkg.NewHiveotWssServer(device.HttpServer, 0)
	}
	err = device.TransportServer.Start()
	if err != nil {
		device.HttpServer.Stop()
		device.HttpServer = nil
		device.TransportServer = nil
		return err
	}
	// populate the security and forms in the TD
	device.TransportServer.AddTDSecForms(device.td, true)
	// create the agent and link it to the transport to serve requests
	device.Agent = agent.NewAgent(device.agentID, nil)
	device.TransportServer.SetRequestSink(device.Agent.HandleRequest)
	// device does ignores connection notifications
	device.TransportServer.SetNotificationSink(func(*msg.NotificationMessage) { /*dummy*/ })
	device.Agent.SetNotificationSink(device.TransportServer.SendNotification)
	// this device envelope handles requests received via the agent
	// device.Agent.SetRequestSink(device.HandleRequest)
	return nil
}

// shutdown the server
func (device *TestDevice) Stop() {
	if device.TransportServer != nil {
		device.TransportServer.Stop()
	}
	if device.HttpServer != nil {
		device.HttpServer.Stop()
	}
}

// NewTestDevice creates a test device containing a transport server linked to an agent.
// The provided authenticator is used to authenticate requests.
//
// Use the agent hooks to handle requests and publish notifications.
//
// To ignore authentication, set the ValidateTokenHandler in httpserver config to a
// function that always returns true.
//
// # The device ThingID will be set to the agentID
//
// cfg defines the server setup.
// agentID is the device agent/thing
// authenticator is used by the test device to authenticate requests
// tm is the TM of the thing to manage
// protocolType sets the type of server to use
func NewTestDevice(cfg *httptransport.Config, agentID string,
	authenticator transport.IAuthenticator, tm *td.TD, protocolType string) *TestDevice {

	v := &TestDevice{
		authenticator: authenticator,
		protocolType:  protocolType,
		agentID:       agentID,
		cfg:           cfg,
		td:            tm,
	}
	return v
}

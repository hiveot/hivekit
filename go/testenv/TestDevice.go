package testenv

import (
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/clients"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/modules/transports/httpbasic"
	"github.com/hiveot/hivekit/go/modules/transports/httpserver"
	httpserverconfig "github.com/hiveot/hivekit/go/modules/transports/httpserver/config"
	ssetransport "github.com/hiveot/hivekit/go/modules/transports/ssesc"
	wsstransport "github.com/hiveot/hivekit/go/modules/transports/wss"
)

// TestDevice contains a server and agent for testing and simulation
type TestDevice struct {
	modules.HiveModuleBase

	agentID         string
	cfg             *httpserverconfig.Config
	HttpServer      transports.IHttpServer
	TransportServer transports.ITransportServer
	Agent           *clients.Agent
	// the server protocol to use, eg ProtocolTypeWotWSS, ...
	protocolType string

	td *td.TD
}

// return the TD of the test device
func (device *TestDevice) GetTD() *td.TD {
	return device.td
}

// Start the test device
// This starts the http server, the messaging transport (sub-protocol) and
// creates an agent instance.
func (device *TestDevice) Start() error {
	device.SetModuleID(device.agentID)

	// setup the server, transport and link the device to the transport
	// cfg := httpserverapi.NewConfig(addr, port, serverCert, caCert, validateToken)
	device.HttpServer = httpserver.NewHttpServerModule(device.cfg)
	err := device.HttpServer.Start()
	if err != nil {
		device.HttpServer = nil
		return err
	}
	switch device.protocolType {
	case transports.ProtocolTypeWotHttpBasic:
		device.TransportServer = httpbasic.NewHttpBasicServer(device.HttpServer)
	case transports.ProtocolTypeHiveotSsesc:
		device.TransportServer = ssetransport.NewSseScServer(device.HttpServer, 0)
	case transports.ProtocolTypeWotWebsocket:
		device.TransportServer = wsstransport.NewWotWssServer(device.HttpServer, 0)
	case transports.ProtocolTypeHiveotWebsocket:
		device.TransportServer = wsstransport.NewHiveotWssServer(device.HttpServer, 0)
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
	device.Agent = clients.NewAgent(device.agentID, nil)
	device.TransportServer.SetRequestSink(device.Agent.HandleRequest)
	device.Agent.SetNotificationSink(device.TransportServer.SendNotification)
	// this device envelope handles requests received via the agent
	device.Agent.SetRequestSink(device.HandleRequest)
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

// NewTestDevice creates a test device containing an agent, device and transport server
// This itself is a module that can be used as any other module.
// Use the agent hooks to handle requests and publish notifications.
//
// To ignore authentication, set the ValidateTokenHandler in httpserver config to a
// function that always returns true.
//
// # The module ID will be set to the agentID
//
// cfg defines the server setup.
// agentID is the service that manages the device
// tm is the TM of the thing to manage
// protocolType sets the type of server to use
func NewTestDevice(cfg *httpserverconfig.Config, agentID string, tm *td.TD,
	protocolType string) *TestDevice {
	v := &TestDevice{
		protocolType: protocolType,
		agentID:      agentID,
		cfg:          cfg,
		td:           tm,
	}
	return v
}

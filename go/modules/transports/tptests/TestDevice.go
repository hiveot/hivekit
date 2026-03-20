package tptests

import (
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/clients"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/modules/transports/httpserver"
	httpserverapi "github.com/hiveot/hivekit/go/modules/transports/httpserver/api"
	"github.com/hiveot/hivekit/go/modules/transports/wss"
	"github.com/hiveot/hivekit/go/wot/td"
)

// TestDevice contains a server and agent for testing and simulation
type TestDevice struct {
	modules.HiveModuleBase

	agentID         string
	cfg             *httpserverapi.Config
	HttpServer      transports.IHttpServer
	TransportServer transports.ITransportServer
	Agent           *clients.Agent

	td *td.TD
}

// return the TD of the test device
func (v *TestDevice) GetTD() *td.TD {
	return v.td
}

// Start the test device
// This starts the http server, the messaging transport (sub-protocol) and
// creates an agent instance.
func (v *TestDevice) Start(_ string) error {
	v.SetModuleID(v.agentID)

	// setup the server, transport and link the device to the transport
	// cfg := httpserverapi.NewConfig(addr, port, serverCert, caCert, validateToken)
	v.HttpServer = httpserver.NewHttpServerModule(v.cfg)
	err := v.HttpServer.Start()
	if err != nil {
		v.HttpServer = nil
		return err
	}
	v.TransportServer = wss.NewWotWssTransport(v.HttpServer, 0)
	err = v.TransportServer.Start("")
	if err != nil {
		v.HttpServer.Stop()
		v.HttpServer = nil
		v.TransportServer = nil
		return err
	}
	// populate the forms in the TD
	v.TransportServer.AddTDForms(v.td, true)
	// create the agent and link it to the transport to serve requests
	v.Agent = clients.NewAgent(v.agentID, nil)
	v.TransportServer.SetRequestSink(v.Agent.HandleRequest)
	v.Agent.SetNotificationSink(v.TransportServer.SendNotification)
	return nil
}

// shutdown the server
func (v *TestDevice) Stop() {
	if v.TransportServer != nil {
		v.TransportServer.Stop()
	}
	if v.HttpServer != nil {
		v.HttpServer.Stop()
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
func NewTestDevice(cfg *httpserverapi.Config, agentID string, tm *td.TD) *TestDevice {
	v := &TestDevice{
		agentID: agentID,
		cfg:     cfg,
		td:      tm,
	}
	return v
}

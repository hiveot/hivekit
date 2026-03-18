package router_test

import (
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/modules/transports/httpserver"
	httpserverapi "github.com/hiveot/hivekit/go/modules/transports/httpserver/api"
	"github.com/hiveot/hivekit/go/modules/transports/wss"
	"github.com/hiveot/hivekit/go/vocab"
	"github.com/hiveot/hivekit/go/wot"
	"github.com/hiveot/hivekit/go/wot/td"
)

// Virtual device contains a server and agent for testing and simulation
type VirtualDevice struct {
	modules.HiveModuleBase

	cfg             *httpserverapi.Config
	httpServer      transports.IHttpServer
	transportServer transports.ITransportServer

	thingID string
}

// Return the TD of the device
func (v *VirtualDevice) GetTD() *td.TD {
	tdi := td.NewTD(v.GetModuleID(), v.thingID, "virtual device", vocab.ThingActuator)
	tdi.AddProperty(vocab.PropSwitch, "on/off status", "", wot.DataTypeBool)
	return tdi
}

// start the test device
func (v *VirtualDevice) Start(_ string) error {
	// setup the server, transport and link the device to the transport
	// cfg := httpserverapi.NewConfig(addr, port, serverCert, caCert, validateToken)
	v.httpServer = httpserver.NewHttpServerModule(v.cfg)
	err := v.httpServer.Start()
	if err != nil {
		v.httpServer = nil
		return err
	}
	v.transportServer = wss.NewWotWssTransport(v.httpServer, 0)
	err = v.transportServer.Start("")
	if err != nil {
		v.httpServer.Stop()
		v.httpServer = nil
		v.transportServer = nil
		return err
	}
	return nil
}

// shutdown the server
func (v *VirtualDevice) Stop() {
	if v.transportServer != nil {
		v.transportServer.Stop()
	}
	if v.httpServer != nil {
		v.httpServer.Stop()
	}
}

// StartVirtualDevice creates a dummy device agent and server
func NewVirtualDevice(cfg *httpserverapi.Config, thingID string) *VirtualDevice {
	v := &VirtualDevice{
		cfg:     cfg,
		thingID: thingID,
	}
	return v
}

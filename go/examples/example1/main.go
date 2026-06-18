package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/examples/example1/counterdevice"
	"github.com/hiveot/hivekit/go/modules/certs"
	certspkg "github.com/hiveot/hivekit/go/modules/certs/pkg"
	"github.com/hiveot/hivekit/go/modules/factory"
	factorypkg "github.com/hiveot/hivekit/go/modules/factory/pkg"
	"github.com/hiveot/hivekit/go/modules/transport"
	"github.com/hiveot/hivekit/go/modules/transport/addforms"
	addformspkg "github.com/hiveot/hivekit/go/modules/transport/addforms/pkg"
	"github.com/hiveot/hivekit/go/modules/transport/discovery"
	discoverypkg "github.com/hiveot/hivekit/go/modules/transport/discovery/pkg"
	tlsserverpkg "github.com/hiveot/hivekit/go/modules/transport/tlsserver/pkg"
	"github.com/hiveot/hivekit/go/modules/transport/wss"
	wsspkg "github.com/hiveot/hivekit/go/modules/transport/wss/pkg"
	"github.com/hiveot/hivekit/go/utils"
)

// Module chain for a device that provides a standalone server for a counter module.
var moduleChain = []factory.ModuleDefinition{
	// create a self-signed CA and server cert if none exist
	{
		Type:        certs.InitFactoryCertsModuleType,
		Constructor: certspkg.NewInitFactoryCerts,
	},
	// http server module is used by websockets
	{Type: transport.TLSServerModuleType,
		Constructor: tlsserverpkg.NewTLSServerFactory,
	},
	// websockets is the main communication transport
	{Type: wss.HiveotWebsocketServerModuleType,
		Constructor: wsspkg.NewHiveotWssServerFactory,
	},
	// counter is the application module
	{Type: counterdevice.CounterDeviceModuleType,
		Constructor: counterdevice.MyCounterModuleFactory,
	},
	// add forms to published TDs
	{
		Type:        addforms.AddFormsModuleType,
		Constructor: addformspkg.NewAddFormsServiceFactory,
	},
	// discovery server for publishing the counter TD
	{Type: discovery.DiscoveryServerModuleType,
		Constructor: discoverypkg.NewDiscoveryServerFactory,
	},
}

func main() {
	utils.SetLogging("info", "")
	// start the factory using the default installation home directory
	env := factory.NewAppEnvironment("~/bin/hiveot", true)
	env.RpcTimeout = time.Minute

	f := factorypkg.NewModuleFactory(env, nil)
	f.SetAuthenticator(nil) // disable auth
	err := f.Start()
	if err != nil {
		slog.Error("Startup failed")
		return
	}

	// run it with the recipe
	r := factorypkg.NewChainRecipe(f, moduleChain)
	r.Start()

	// increment the counter to generate an event
	m, err := f.GetModule(counterdevice.CounterDeviceModuleType, false)
	req := msg.NewRequestMessage(td.OpInvokeAction,
		counterdevice.DefaultCounterDeviceThingID,
		counterdevice.IncrementActionName, nil)
	_ = m.HandleRequest(req, nil)

	fmt.Printf("Counter is running and listening on '%s'\n", f.GetConnectURL())
	fmt.Printf("Use the cli from example 2 to read its status\n")
	f.WaitForSignal(context.Background())
	f.Stop()
}

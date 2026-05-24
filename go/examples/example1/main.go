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
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/modules/transports/addforms"
	addformspkg "github.com/hiveot/hivekit/go/modules/transports/addforms/pkg"
	"github.com/hiveot/hivekit/go/modules/transports/discovery"
	discoverypkg "github.com/hiveot/hivekit/go/modules/transports/discovery/pkg"
	httptransportpkg "github.com/hiveot/hivekit/go/modules/transports/httptransport/pkg"
	"github.com/hiveot/hivekit/go/modules/transports/wss"
	wsspkg "github.com/hiveot/hivekit/go/modules/transports/wss/pkg"
	"github.com/hiveot/hivekit/go/utils"
)

// Recipe for a simple device that provides a standalone server for a counter module.
var recipe = factorypkg.FactoryRecipe{
	ModuleDefs: map[string]factory.ModuleDefinition{
		// add forms to published TDs
		addforms.AddFormsModuleType: {
			Constructor: addformspkg.NewAddFormsServiceFactory,
		},
		// create a self-signed CA and server cert if none exist
		certs.InitFactoryCertsModuleType: {
			Constructor: certspkg.NewInitFactoryCerts,
		},
		// discovery server for publishing the counter TD
		discovery.DiscoveryServerModuleType: {
			Constructor: discoverypkg.NewDiscoveryServerFactory,
		},
		// http server module is used by websockets
		transports.HttpServerModuleType: {
			Constructor: httptransportpkg.NewHttpTransportServerFactory,
		},
		// websockets is the main communication transport
		wss.HiveotWebsocketServerModuleType: {
			Constructor: wsspkg.NewHiveotWssServerFactory,
		},
		// counter is the application module
		counterdevice.CounterDeviceModuleType: {
			Constructor: counterdevice.MyCounterModuleFactory,
		},
	},

	// This chain defines the complete application
	ModuleChain: []string{
		// initialize the environment with certificates
		certs.InitFactoryCertsModuleType,
		// run a websocket server (this loads the http transport)
		wss.HiveotWebsocketServerModuleType,
		// run the counter service
		counterdevice.CounterDeviceModuleType,
		// add forms to the counter TD
		addforms.AddFormsModuleType,
		// run discovery server to publish the counter TD
		discovery.DiscoveryServerModuleType,
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
	r := factorypkg.NewFactoryRecipe(recipe.ModuleDefs, recipe.ModuleChain)
	r.Start(f)

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

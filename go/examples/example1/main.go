package main

import (
	"context"
	"fmt"

	"github.com/hiveot/hivekit/go/examples/example1/counterdevice"
	"github.com/hiveot/hivekit/go/modules/certs"
	certspkg "github.com/hiveot/hivekit/go/modules/certs/pkg"
	"github.com/hiveot/hivekit/go/modules/factory"
	factorypkg "github.com/hiveot/hivekit/go/modules/factory/pkg"
	"github.com/hiveot/hivekit/go/modules/transports"
	httptransportpkg "github.com/hiveot/hivekit/go/modules/transports/httptransport/pkg"
	"github.com/hiveot/hivekit/go/modules/transports/wss"
	wsspkg "github.com/hiveot/hivekit/go/modules/transports/wss/pkg"
	"github.com/hiveot/hivekit/go/utils"
)

// Recipe for a simple device that provides a standalone server for a counter module.
var recipe = factorypkg.FactoryRecipe{
	ModuleDefs: map[string]factory.ModuleDefinition{
		// create a self-signed CA and server cert if none exist
		certs.InitFactoryCertsModuleType: {
			Constructor: certspkg.NewInitFactoryCerts,
		},
		transports.HttpServerModuleType: {
			Constructor: httptransportpkg.NewHttpTransportServerFactory,
		},
		wss.HiveotWebsocketServerModuleType: {
			Constructor: wsspkg.NewHiveotWssServerFactory,
		},
		wss.HiveotWebsocketClientModuleType: {
			Constructor: wsspkg.NewHiveotWssClientFactory,
		},
		counterdevice.CounterDeviceModuleType: {
			Constructor: counterdevice.MyCounterModuleFactory,
		},
	},

	// This chain defines the complete application
	ModuleChain: []string{
		certs.InitFactoryCertsModuleType,
		wss.HiveotWebsocketServerModuleType,
		counterdevice.CounterDeviceModuleType,
	},
}

func main() {
	utils.SetLogging("info", "")
	// start the factory using the default installation home directory
	env := factory.NewAppEnvironment("~/bin/hiveot", true)
	f := factorypkg.NewModuleFactory(env, nil)
	err := f.Start()
	if err == nil {
		// run it with the recipe
		r := factorypkg.NewFactoryRecipe(recipe.ModuleDefs, recipe.ModuleChain)
		_ = r
		r.Start(f)
	}
	fmt.Printf("Counter is running and listening on '%s'\n", f.GetConnectURL())
	fmt.Printf("Use the cli from example 2 to read its status\n")
	f.WaitForSignal(context.Background())
	f.Stop()
}

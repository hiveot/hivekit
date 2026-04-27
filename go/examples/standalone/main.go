package main

import (
	"github.com/hiveot/hivekit/go/examples/standalone/counterdevice"
	"github.com/hiveot/hivekit/go/modules/factory"
	factorypkg "github.com/hiveot/hivekit/go/modules/factory/pkg"
	"github.com/hiveot/hivekit/go/modules/transports/wss"
	wsspkg "github.com/hiveot/hivekit/go/modules/transports/wss/pkg"
)

// Recipe for a simple device that provides a standalone server for a counter module.
var recipe = factorypkg.FactoryRecipe{
	ModuleDefs: map[string]factory.ModuleDefinition{
		// transports.HttpServerModuleType: {
		// Constructor: httpserver.NewHttpServerFactory,
		// },
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
		// wss.HiveotWebsocketServerModuleType,
		counterdevice.CounterDeviceModuleType,
	},
}

func main() {
	// start the factory
	env := factory.NewAppEnvironment("", true)
	f := factorypkg.NewModuleFactory(env, nil)
	f.Start()
	// run it with the recipe
	r := factorypkg.NewFactoryRecipe(recipe.ModuleDefs, recipe.ModuleChain)
	_ = r
	r.Start(f)
}

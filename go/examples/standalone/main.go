package main

import (
	"github.com/hiveot/hivekit/go/examples/standalone/counterdevice"
	"github.com/hiveot/hivekit/go/modules/factory"
	factoryrecipe "github.com/hiveot/hivekit/go/modules/factory/recipe"
	wss "github.com/hiveot/hivekit/go/modules/transports/wss1"
	wsspkg "github.com/hiveot/hivekit/go/modules/transports/wss1/pkg"
)

// Recipe for a simple device that provides a standalone server for a counter module.
var StandaloneDeviceRecipe = factoryrecipe.FactoryRecipe{
	ModuleDefs: map[string]factory.ModuleDefinition{
		wss.HiveotWebsocketClientModuleType: {
			Constructor: wsspkg.NewHiveotWssClientFactory,
		},
	},
	// This chain defines the complete application
	ModuleChain: []string{
		wss.HiveotWebsocketServerModuleType,
		counterdevice.CounterDeviceModuleType,
	},
}

func main() {
	println("Hello world")
}

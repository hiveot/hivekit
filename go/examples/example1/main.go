package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/hiveot/hivekit/go/api"

	factorypkg "github.com/hiveot/hivekit/go/modules/factory/pkg"
	"github.com/hiveot/hivekit/go/modules/factory/recipes"
	"github.com/hiveot/hivekit/go/testenv"
	"github.com/hiveot/hivekit/go/utils"
)

// Demo stand-alone IoT device running the test counting device
// This uses the Device-Server recipe and inserts the test counter module into the app slot.
// See the factory/recipes/DeviceServerRecipe.go for the modules in the recipe.
// On start the device publishes its TD to the discovery server.
func main() {
	utils.SetLogging("info", "")
	// start the factory using the default installation home directory
	env := api.NewAppEnvironment("~/bin/hiveot", true)
	env.RpcTimeout = time.Minute

	f := factorypkg.NewModuleFactory(env, nil)
	f.SetAuthenticator(nil) // disable auth

	// Define the counter test device for use with the factory recipe
	appDef := &api.ModuleDefinition{
		Type:        env.AppID,
		Constructor: testenv.NewCounterDeviceFactory,
		Config: &testenv.CounterConfig{
			AutoIncrement: false,
			ResetValue:    60,
		},
	}
	// the device server recipe contains modules for running a server with certs and authn
	// you can message the recipe as a module or via a client. Here we message directly.
	r := recipes.NewStandAloneDeviceRecipe(f, appDef)

	err := r.Start()
	if err != nil {
		fmt.Println("Startup failed: " + err.Error())
		os.Exit(1)
	}

	fmt.Printf("main: Counter is running and listening on '%s'\n", f.GetConnectURL())
	fmt.Printf("main: Use the cli from example 2 to read its status\n")
	f.WaitForSignal(context.Background())
	f.Stop()
}

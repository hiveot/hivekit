package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/examples/example1/counterdevice"
	"github.com/hiveot/hivekit/go/modules/factory"
	factorypkg "github.com/hiveot/hivekit/go/modules/factory/pkg"
	"github.com/hiveot/hivekit/go/modules/factory/recipes"
	"github.com/hiveot/hivekit/go/utils"
)

// Demo IoT device running a counter server
// This uses the Device-Server recipe and inserts the counter module into the app slot.
// See the factory/recipes/DeviceServerRecipe.go for the modules in the recipe.
// On start the device publishes its TD to the discovery server.
func main() {
	utils.SetLogging("info", "")
	// start the factory using the default installation home directory
	env := factory.NewAppEnvironment("~/bin/hiveot", true)
	env.RpcTimeout = time.Minute

	f := factorypkg.NewModuleFactory(env, nil)
	f.SetAuthenticator(nil) // disable auth

	// run it with the recipe
	appDef := &factory.ModuleDefinition{
		Type:        env.AppID,
		Constructor: counterdevice.MyCounterModuleFactory,
	}
	r := recipes.NewDeviceServerRecipe(f, appDef)
	// r := factorypkg.NewChainRecipe(f, moduleChain)
	err := r.Start()
	if err != nil {
		fmt.Println("Startup failed: " + err.Error())
		os.Exit(1)
	}
	// increment the counter using a message
	req := msg.NewRequestMessage(td.OpInvokeAction,
		counterdevice.DefaultCounterDeviceThingID,
		counterdevice.IncrementActionName, nil)
	req.SenderID = "main"
	_ = r.HandleRequest(req, func(*msg.ResponseMessage) error { return nil })

	fmt.Printf("Counter is running and listening on '%s'\n", f.GetConnectURL())
	fmt.Printf("Use the cli from example 2 to read its status\n")
	f.WaitForSignal(context.Background())
	f.Stop()
}

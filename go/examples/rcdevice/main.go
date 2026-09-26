package main

import (
	"context"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/utils"

	rcdevice_recipe "github.com/hiveot/hivekit/go/factory/recipes/rcdevice"
	"github.com/hiveot/hivekit/go/testenv"
)

const DeviceClientID = "admin"

var ExampleHome = path.Join(os.TempDir(), "hivekit-examples")

// Demo reverse-connection IoT device running the test counting device.
//
// This uses the "RCDeviceRecipe" factory recipe.
//
// See the factory/recipes/rcdevice/RCDeviceRecipe.go for the cells in the recipe.
// On start the device publishes its TD to the discovery server.
func main() {
	env := api.NewHiveEnvironment(ExampleHome, true)
	env.AppID = "example-1"
	env.RpcTimeout = time.Minute // for testing
	env.HttpsPort = 9222         // for testing
	if env.ClientID == "" {
		env.ClientID = api.DefaultAdminUserID
	}
	utils.SetLogging(env.LogLevel, "")

	// start the app service
	cfg := &testenv.CounterConfig{
		AutoIncrement: false,
		ResetValue:    60,
	}
	counterThing, err := testenv.NewTestCounterThing(env.AppID, cfg)

	// link to it from the RC device recipe
	// the RC device recipe contains cells for connecting to a gateway.
	r, f, err := rcdevice_recipe.NewRCDeviceRecipe(env, counterThing)
	if err != nil {
		fmt.Println("Startup failed: " + err.Error())
		os.Exit(1)
	}
	// signal the app is ready to go and all cells are linked
	r.Start()
	defer r.Stop()
	counterThing.Start()

	fmt.Printf("main: homeDir: %s\n", env.HomeDir)
	fmt.Printf("main: Counter is running and listening on '%v'\n", f.GetConnectURLs())
	fmt.Printf("main: Use the cli from example 2 to read its status\n")
	f.WaitForSignal(context.Background())
}

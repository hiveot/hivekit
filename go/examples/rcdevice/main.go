package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path"
	"time"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/utils"

	rcdevice_recipe "github.com/hiveot/hivekit/go/factory/recipes/rcdevice"
	"github.com/hiveot/hivekit/go/testenv"
)

const DeviceClientID = "counter1"

var ExampleHome = path.Join(os.TempDir(), "hivekit-examples")

// Demo reverse-connection IoT device running the test counting device.
//
// This discovers and connects to the gateway. A client certficiate for the device
// must be created first using the cli (cli createCert counter1)
//
// This uses the "RCDeviceRecipe" factory recipe.
//
// See the factory/recipes/rcdevice/RCDeviceRecipe.go for the cells in the recipe.
// On start the device publishes its TD to the discovery server.
func main() {
	var err error
	env := api.NewHiveEnvironment(ExampleHome, true)
	env.AppID = "rcdevice-example"

	env.RpcTimeout = time.Minute // for testing
	env.HttpsPort = 9222         // for testing
	if env.ClientID == "" {
		env.ClientID = DeviceClientID
	}
	utils.SetLogging(env.LogLevel, "")

	// this requires a client certificate for authentication with the gateway
	_, err = env.GetClientCert()
	if err != nil {
		err := fmt.Errorf("This example needs a client certificate for device '%s'. Use the cli to create it.", env.ClientID)
		slog.Error(err.Error())
		os.Exit(1)
	}

	// create the test counter cell
	cfg := &testenv.CounterConfig{
		AutoIncrement: false,
		ResetValue:    60,
	}
	counterThing, err := testenv.NewTestCounterThing(env.ClientID, cfg)

	// link it to the RC device recipe that handles connections.
	r, f, err := rcdevice_recipe.NewRCDeviceRecipe(env, counterThing)
	if err != nil {
		fmt.Println("Startup failed: " + err.Error())
		os.Exit(1)
	}
	// signal the app is ready to go and all cells are linked
	r.Start()
	defer r.Stop()
	counterThing.Start()
	rcr := r.GetClient()

	fmt.Printf("main: homeDir: %s\n", env.HomeDir)
	fmt.Printf("main: Counter is running and connected as Thing '%s' to server at %v\n",
		counterThing.GetID(), rcr.GetConnectionURL())
	fmt.Printf("main: Use the cli from example 2 to read its status\n")
	f.WaitForSignal(context.Background())
}

package main

import (
	"context"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/utils"

	sadevice_recipe "github.com/hiveot/hivekit/go/factory/recipes/sadevice"
	factory_service "github.com/hiveot/hivekit/go/factory/service"
	"github.com/hiveot/hivekit/go/testenv"
)

// Create a client account to login as.
// By convention this creates a token in {home}/certs/{clientID}.token
const ExampleClientID = "admin"

var ExampleHome = path.Join(os.TempDir(), "hivekit-examples")

// Demo stand-alone IoT device running the test counting device.
//
// This uses the "StandAloneDevice" factory recipe and inserts the test counter service
// into the app slot.
//
// The factory authn service factory creates an admin client certificate and auth token if
// not present.
//
// See the factory/recipes/standalone/StandAloneDeviceRecipe.go for the cells in the recipe.
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

	// Link the stand-alone recipe to the counter device.
	// The recipe contains cells for running a server with certs and authn.
	f := factory_service.NewCellFactory(env, nil)
	r, err := sadevice_recipe.NewStandAloneDeviceRecipe(f, counterThing)
	_ = r
	if err != nil {
		fmt.Println("Startup failed: " + err.Error())
		os.Exit(1)
	}
	// signal the app is ready to go and all cells are linked
	f.Start()
	counterThing.Start()

	fmt.Printf("main: homeDir: %s\n", env.HomeDir)
	fmt.Printf("main: Counter is running and listening on '%v'\n", f.GetConnectURLs())
	fmt.Printf("main: Use the cli from example 2 to read its status\n")
	f.WaitForSignal(context.Background())
	f.Stop()
}

package main

import (
	"context"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/hiveot/hivekit/go/api"

	standalonerecipe "github.com/hiveot/hivekit/go/modules/factory/recipes/standalone"
	factory_service "github.com/hiveot/hivekit/go/modules/factory/service"
	"github.com/hiveot/hivekit/go/testenv"
)

// Create a client account to login as.
// By convention this creates a token in {home}/certs/{clientID}.token
const ExampleClientID = "admin"

var ExampleHome = path.Join(os.TempDir(), "hivekit-examples")

// Demo stand-alone IoT device running the test counting device.
//
// This uses the "StandAloneDevice" factory recipe and inserts the test counter module
// into the app slot.
//
// The factory authn module factory creates an admin client certificate and auth token if
// not present.
//
// See the factory/recipes/StandAloneDeviceRecipe.go for the modules in the recipe.
// On start the device publishes its TD to the discovery server.
func main() {
	env := api.NewAppEnvironment(ExampleHome, true)
	env.RpcTimeout = time.Minute // for testing
	env.HttpsPort = 9222         // for testing

	if env.ClientID == "" {
		env.ClientID = api.DefaultAdminUserID
	}

	f := factory_service.NewModuleFactory(env, nil)

	// the device server recipe contains modules for running a server with certs and authn
	// you can message the recipe as a module or via a client. Here we message directly.
	r := standalonerecipe.NewStandAloneDeviceRecipe(f)
	err := r.Start()
	if err != nil {
		fmt.Println("Startup failed: " + err.Error())
		os.Exit(1)
	}

	// next start the app module
	cfg := &testenv.CounterConfig{
		AutoIncrement: false,
		ResetValue:    60,
	}
	appModule := testenv.NewCounterDevice("", cfg)

	// requests from the app module are passed to the modules in the chain
	// intended to publish the TD. No other requests are expected.
	appModule.SetRequestSink(r) // chain handles requests from the module
	// notifications from the chain are passed to the app, eg connection established
	// not much else to do here
	r.SetNotificationSink(appModule)
	// requests from the chain server are passed to the app module. This is the 'Thing' it serves.
	r.SetRequestSink(appModule)
	// Property and event notifications published by the app are send to connected clients.
	// the recipe HandleNotification passes it to the last module in the chain and up from there.
	appModule.SetNotificationSink(r)
	appModule.Start()

	fmt.Printf("main: homeDir: %s\n", env.HomeDir)
	fmt.Printf("main: Counter is running and listening on '%s'\n", f.GetConnectURL())
	fmt.Printf("main: Use the cli from example 2 to read its status\n")
	f.WaitForSignal(context.Background())
	f.Stop()
}

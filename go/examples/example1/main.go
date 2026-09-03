package main

import (
	"context"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/utils"

	standalonerecipe "github.com/hiveot/hivekit/go/factory/recipes/standalone"
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
// See the factory/recipes/StandAloneDeviceRecipe.go for the cells in the recipe.
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

	f := factory_service.StartCellFactory(env, nil)

	// the device server recipe contains cells for running a server with certs and authn
	// you can message the recipe as a service or via a client. Here we message directly.
	r := standalonerecipe.NewStandAloneDeviceRecipe(f)
	err := r.Start()
	if err != nil {
		fmt.Println("Startup failed: " + err.Error())
		os.Exit(1)
	}

	// next start the app service
	cfg := &testenv.CounterConfig{
		AutoIncrement: false,
		ResetValue:    60,
	}
	counterThing, err := testenv.StartTestCounterThing(env.AppID, cfg)

	// Requests from the counter thing are passed to the cells in the chain.
	// Intended to publish the TD. No other requests are expected.
	counterThing.SetRequestSink(r)
	// Notifications from the counter are passed to the app, eg connection established.
	// Not much else to do here.
	r.SetNotificationSink(counterThing)
	// Requests from the chain are passed to the device. This is the 'Thing' it serves.
	r.SetRequestSink(counterThing)
	// Property and event notifications published by the app are send to connected clients.
	// the recipe HandleNotification passes it to the last cell in the chain and up from there.
	counterThing.SetNotificationSink(r)
	// Start after linking.

	fmt.Printf("main: homeDir: %s\n", env.HomeDir)
	fmt.Printf("main: Counter is running and listening on '%v'\n", f.GetConnectURLs())
	fmt.Printf("main: Use the cli from example 2 to read its status\n")
	f.WaitForSignal(context.Background())
	f.Stop()
}

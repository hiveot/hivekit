package main

import (
	"context"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/hiveot/hivekit/go/api"

	"github.com/hiveot/hivekit/go/modules/authn"
	factorypkg "github.com/hiveot/hivekit/go/modules/factory/pkg"
	"github.com/hiveot/hivekit/go/modules/factory/recipes"
	"github.com/hiveot/hivekit/go/testenv"
	"github.com/hiveot/hivekit/go/utils"
)

// Create an admin account to login as
const ExampleClientID = "admin"

var ExampleHome = path.Join(os.TempDir(), "hivekit-examples")

// Demo stand-alone IoT device running the test counting device.
//
// This uses the "StandAloneDevice" factory recipe and inserts the test counter module
// into the app slot.
//
// See the factory/recipes/StandAloneDeviceRecipe.go for the modules in the recipe.
// On start the device publishes its TD to the discovery server.
func main() {
	utils.SetLogging("info", "")
	// start the factory using the examples tmp directory as home
	env := api.NewAppEnvironment(ExampleHome, true)
	env.RpcTimeout = time.Minute

	f := factorypkg.NewModuleFactory(env, nil)

	// the device server recipe contains modules for running a server with certs and authn
	// you can message the recipe as a module or via a client. Here we message directly.
	r := recipes.NewStandAloneDeviceRecipe(f)
	err := r.Start()
	if err != nil {
		fmt.Println("Startup failed: " + err.Error())
		os.Exit(1)
	}

	// Create an example operator account for the client and export its admin token.
	// both login and password are ExampleClientID ("example1")
	// FIXME: would this be better for the factory or authn service?
	authnSvc := api.GetFactoryModule[authn.IAuthnService](f, authn.AuthnServiceModuleType)
	if authnSvc != nil {
		var token string
		_ = authnSvc.AddClient(ExampleClientID, "Example client", authn.ClientRoleOperator)
		_ = authnSvc.SetPassword(ExampleClientID, ExampleClientID)
		// this client test token can be used for an hour
		clientID := ExampleClientID
		token, _, err = authnSvc.GetSessionManager().CreateToken(clientID, time.Hour*24)
		tokenFile := path.Join(env.CertsDir, clientID+".token")
		err := os.MkdirAll(env.CertsDir, 0700)
		if err != nil {
			fmt.Printf("main:ERROR creating certs dir: %s\n", err.Error())
			os.Exit(1)
		}
		err = os.Remove(tokenFile)
		err = os.WriteFile(tokenFile, []byte(token), 0400)
		if err != nil {
			fmt.Printf("main:ERROR writing auth token: %s\n", err.Error())
			os.Exit(1)
		}
		fmt.Printf("Created new admin token at '%s'\n", tokenFile)
	}

	// next start the app module
	cfg := &testenv.CounterConfig{
		AutoIncrement: false,
		ResetValue:    60}
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

	fmt.Printf("main: Counter is running and listening on '%s'\n", f.GetConnectURL())
	fmt.Printf("main: Use the cli from example 2 to read its status\n")
	f.WaitForSignal(context.Background())
	f.Stop()
}

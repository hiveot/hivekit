package main

import (
	"context"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/hiveot/hivekit/go/api"
	gatewayrecipe "github.com/hiveot/hivekit/go/cells/factory/recipes/gateway"
	factory_service "github.com/hiveot/hivekit/go/cells/factory/service"
	"github.com/hiveot/hivekit/go/utils"
)

var ExampleHome = path.Join(os.TempDir(), "hivekit-examples")

// Example of an IoT gateway using the HiveKit gateway factory recipe.
//
// The authn service factory creates a new admin auth token if not present.
// The certs service factory creates a new admin client cert if not present.
//
// See the factory/recipes/GatewayRecipe.go for the cells in the recipe.
func main() {

	env := api.NewHiveEnvironment(ExampleHome, true)
	env.RpcTimeout = time.Minute // for testing
	utils.SetLogging(env.LogLevel, "")

	f := factory_service.NewCellFactory(env, nil)
	r := gatewayrecipe.NewGatewayDeviceRecipe(f, false)

	// the authn factory creates an admin token and client certificate for use by consumers

	err := r.Start()
	if err != nil {
		fmt.Println("Startup failed: " + err.Error())
		os.Exit(1)
	}

	fmt.Printf("main: homeDir: %s\n", env.HomeDir)
	fmt.Printf("main: Counter is running and listening on '%s'\n", f.GetConnectURL())
	f.WaitForSignal(context.Background())
	f.Stop()

}

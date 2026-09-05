package main

import (
	"fmt"
	"os"
	"path"
	"time"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/cells/router"
	"github.com/hiveot/hivekit/go/examples/example3/tuiapp"
	consumer_recipe "github.com/hiveot/hivekit/go/factory/recipes/consumer"
	factory_service "github.com/hiveot/hivekit/go/factory/service"
	"github.com/hiveot/hivekit/go/utils"
)

// Use the admin account to login as. This uses the home/certs directory to load the token.
const ExampleClientID = "admin"

var ExampleHome = path.Join(os.TempDir(), "hivekit-examples")

func main() {

	env := api.NewHiveEnvironment(ExampleHome, true)
	env.AppID = "example-3"
	env.RpcTimeout = time.Second * 60 // avoid comm timeout during debugging
	// FIXME: for a different clientID when running with go run, instead of the APP ID
	if env.ClientID == "main" {
		env.ClientID = "admin"
	}

	// utils.SetLogging("warn", "")
	// log to file to avoid messing up the tui
	env.CreateDir(env.LogsDir, 0750)
	utils.SetLogging(env.LogLevel, path.Join(env.LogsDir, "example3.log"))

	f := factory_service.StartCellFactory(env, nil)
	r, err := consumer_recipe.StartConsumerRecipe(f, false)
	if err != nil {
		os.Exit(1)
	}

	// Set default credentials for connecting to devices with the router service.
	// The router looks up the credentials for connecting to standalone devices using
	// the device thingID and falls back to the "" thingID.
	authToken, _ := env.GetAuthToken()
	rtr := api.GetFactoryCell[router.IRouterService](f, router.RouterCellType)
	rtr.AddDeviceCredential("", env.ClientID, authToken, td.SecSchemeBearer)
	fmt.Printf("Using '%s' as login ID\n", env.ClientID)

	app := tuiapp.NewTuiApp(f)
	app.SetRequestSink(r)
	r.SetNotificationSink(app)
	// signal the app is ready to go and all cells are linked
	r.Ready()

	app.Start()
	if err != nil {
		println("Tui failed to start: ", err.Error())
	} else {
		println("Done\n")
	}
}

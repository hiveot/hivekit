package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/examples/example2/cliex"
	"github.com/hiveot/hivekit/go/modules/directory"
	factorypkg "github.com/hiveot/hivekit/go/modules/factory/pkg"
	"github.com/hiveot/hivekit/go/modules/factory/recipes"
	"github.com/hiveot/hivekit/go/modules/router"
	"github.com/hiveot/hivekit/go/modules/transport/discovery"
)

// Create an admin account to login as
const ExampleClientID = "admin"

var ExampleHome = path.Join(os.TempDir(), "hivekit-examples")

// CLI example commands:
//	cliex  [-txt] discover           discover devices on the network
//	cliex  td  <thingID>             show the TD of a discovered thing
//	cliex  status  <thingID>         show the current status of a thing
//	cliex  subscribe  <thingID>      subscribe to updates of a thing

const (
	CmdDiscover   = "discover"
	CmdListDir    = "dir"
	CmdLogin      = "login"
	CmdShowTD     = "td"
	CmdShowStatus = "status"
	CmdSubscribe  = "subscribe"
)

var appConfig cliex.CliexConfig

// Run the CLI app using the Consumer Recipe
func main() {
	// setup the environment
	env := api.NewAppEnvironment(ExampleHome, true)
	// FIXME: how to set a default clientID instead of the APP ID
	if env.ClientID == "" {
		env.ClientID = "admin"
	}
	env.RpcTimeout = time.Minute * 6 // for testing

	// environment defaults
	flag.BoolVar(&appConfig.Subscribe, "subscribe", appConfig.Subscribe, "Subscribe to events or property changes until ^C")
	flag.BoolVar(&appConfig.Verbose, "v", appConfig.Verbose, "Show more detailed output")
	flag.BoolVar(&appConfig.NoDisco, "nd", appConfig.NoDisco, "Do not start with discovery")

	args := flag.Args()
	if len(args) == 0 {
		fmt.Printf("cliex [options] command  \n\n")
		fmt.Println("Where command is one of:")
		fmt.Printf(" %-10s           Discover WoT devices and directories\n", CmdDiscover)
		fmt.Printf(" %-10s thingID   Login to the device\n", CmdLogin)
		fmt.Printf(" %-10s thingID   List the content of a directory\n", CmdListDir)
		fmt.Printf(" %-10s thingID   Show the TD of a Thing\n", CmdShowTD)
		fmt.Printf(" %-10s thingID   Show the current status of a Thing\n", CmdShowStatus)
		fmt.Printf(" %-10s thingID   Subscribe to Thing events and property updates\n", CmdSubscribe)
		fmt.Println("\nOptions:")
		// flag.Usage()
		flag.PrintDefaults()
		return
	}
	cmd := args[0]

	getArgs := func() string {
		if len(args) > 1 {
			return args[1]
		}
		fmt.Println("\nMissing thingID argument")
		os.Exit(1)
		return ""
	}

	// Ignore the certificate check just for this example. Dont do this in your app.
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	// Start the CLI recipe modules
	f := factorypkg.NewModuleFactory(env, nil)
	r := recipes.NewConsumerRecipe(f)
	err := r.Start()
	if err != nil {
		os.Exit(1)
	}
	// the router module sets the default clientID to AppEnvironment.ClientID
	// the default auth token is set to {clientID}.token
	authToken, _ := env.GetAuthToken()
	rtr := api.GetFactoryModule[router.IRouterService](f, router.RouterModuleType)
	rtr.AddDeviceCredential("", env.GetClientID(), authToken, td.SecSchemeBearer)
	fmt.Printf("Using '%s' as login ID\n", env.GetClientID())

	discoClient := api.GetFactoryModule[discovery.IDiscoveryClient](f, discovery.DiscoveryClientModuleType)
	dirClient := api.GetFactoryModule[directory.IDirectoryClient](f, directory.DirectoryClientModuleType)
	app := cliex.NewCliex(appConfig, discoClient, dirClient, f.GetEnvironment().CaCert)

	app.SetRequestSink(r)
	r.SetNotificationSink(app)
	err = app.Start()

	switch cmd {
	case CmdDiscover:
		app.ShowDiscovery()
	case CmdListDir:
		app.ListDir()
	case CmdShowTD:
		thingID := getArgs()
		app.ShowTD(thingID)
	case CmdShowStatus:
		thingID := getArgs()
		app.ShowStatus(thingID, false)
	case CmdSubscribe:
		thingID := getArgs()
		app.Subscribe(thingID)

	default:
		fmt.Printf("\nUnknown command: %s\n", cmd)
	}
}

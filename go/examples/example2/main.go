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
	"github.com/hiveot/hivekit/go/cells/directory"
	"github.com/hiveot/hivekit/go/cells/router"
	"github.com/hiveot/hivekit/go/cells/transport/discovery"
	"github.com/hiveot/hivekit/go/examples/example2/cliex"
	consumerrecipe "github.com/hiveot/hivekit/go/factory/recipes/consumer"
	factory_service "github.com/hiveot/hivekit/go/factory/service"
	"github.com/hiveot/hivekit/go/utils"
)

// By default use the admin account to login as. This uses the home/certs directory to load the token.
const DefaultClientID = "admin"

var ExampleHome = path.Join(os.TempDir(), "hivekit-examples")

// CLI example commands:
//	cliex  [-txt] discover           discover devices on the network
//	cliex  td  <thingID>             show the TD of a discovered thing
//	cliex  status  <thingID>         show the current status of a thing
//	cliex  subscribe  <thingID>      subscribe to updates of a thing

const (
	CmdDiscover    = "discover"
	CmdListDir     = "ld"
	CmdLogin       = "login"
	CmdShowActions = "actions"
	CmdShowTD      = "td"
	CmdShowStatus  = "status"
	CmdSubscribe   = "subscribe"
)

var appConfig cliex.CliexConfig

// Run the CLI app
// This uses the 'ConsumerRecipe' for discovery, reading a directory and routing requests
// to Things.
func main() {

	// flag.CommandLine.Init("CLI example", flag.ContinueOnError)

	// environment defaults
	flag.BoolVar(&appConfig.Subscribe, "subscribe", appConfig.Subscribe, "Subscribe to events or property changes until ^C")
	flag.BoolVar(&appConfig.Verbose, "v", appConfig.Verbose, "Show more detailed output (loglevel info)")
	flag.BoolVar(&appConfig.NoDisco, "nd", appConfig.NoDisco, "Do not start with discovery")

	// flag.CommandLine.Init("CLI example", flag.ContinueOnError)
	flag.Usage = func() {
		fmt.Println("Usage: cliex [options] Command")
		fmt.Println()
		fmt.Println("Commands:")
		fmt.Printf("  %-10s                Discover WoT devices and directories\n", CmdDiscover)
		// fmt.Printf("  %-10s thingID        Set login ID for the device\n", CmdLogin)
		fmt.Printf("  %-10s thingID        List the content of a directory with optional thingID\n", CmdListDir)
		fmt.Printf("  %-10s thingID        Show the TD of a Thing\n", CmdShowTD)
		fmt.Printf("  %-10s thingID        Show the current status of a Thing\n", CmdShowStatus)
		fmt.Printf("  %-10s thingID        Subscribe to Thing events and property updates\n", CmdSubscribe)
		fmt.Printf("  %-10s thingID [actionName]  Show/Invoke actions\n", CmdShowActions)
		fmt.Println()
		fmt.Println("Options:")
		flag.PrintDefaults()
	}

	// Setup the environment after parsing the commandline
	env := api.NewHiveEnvironment(ExampleHome, true)
	env.AppID = "example-2"
	if appConfig.Verbose {
		env.LogLevel = "info"
	}
	utils.SetLogging(env.LogLevel, "")
	if env.ClientID == "" {
		env.ClientID = DefaultClientID
	}

	env.RpcTimeout = time.Minute * 6 // for testing
	args := flag.Args()
	if len(args) == 0 {
		flag.Usage()
		return
	}
	cmd := args[0]

	getThingID := func() string {
		if len(args) > 1 {
			return args[1]
		}
		fmt.Println("\nMissing thingID argument")
		os.Exit(1)
		return ""
	}

	// Ignore the certificate check just for this example. Dont do this in your app.
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	// Start the CLI recipe cells
	f := factory_service.NewCellFactory(env, nil)
	r := consumerrecipe.NewConsumerRecipe(f, false)
	err := r.Start()
	if err != nil {
		os.Exit(1)
	}

	// Set default credentials for connecting to devices with the router service.
	// The router looks up the credentials for connecting to standalone devices using
	// the device thingID and falls back to the "" thingID.
	authToken, _ := env.GetAuthToken()
	rtr := api.GetFactoryCell[router.IRouterService](f, router.RouterCellType)
	if authToken != "" {
		rtr.AddDeviceCredential("", env.ClientID, authToken, td.SecSchemeBearer)
		fmt.Printf("Found auth token for login as '%s'\n", env.ClientID)
	} else {
		fmt.Printf("No auth token. Using '%s' as login ID\n", env.ClientID)
	}
	clientCert, _ := env.GetClientCert()
	if clientCert != nil {
		fmt.Printf("Found Client cert with clientID '%s'\n", clientCert.Leaf.Subject.CommonName)
	} else {
		fmt.Printf("No client Cert found.\n")
	}

	discoClient := api.GetFactoryCell[discovery.IDiscoveryClient](f, discovery.DiscoveryClientCellType)
	dirClient := api.GetFactoryCell[directory.IDirectoryClient](f, directory.DirectoryClientCellType)
	caCert, err := env.GetCACert()
	app := cliex.NewCliex(appConfig, discoClient, dirClient, caCert)

	app.SetRequestSink(r)
	r.SetNotificationSink(app)
	err = app.Start()

	switch cmd {
	case CmdDiscover, "disco":
		app.ShowDiscovery()
	case CmdListDir:
		// optional directory thingID
		thingID := ""
		if len(args) > 1 {
			thingID = args[1]
		}
		app.ListDir(thingID)
	case CmdShowActions:
		thingID := getThingID()
		actionName := ""
		if len(args) > 2 {
			actionName = args[2]
		}
		// providing a name to invoke the action
		app.ShowActions(thingID, actionName)
	case CmdShowTD:
		thingID := getThingID()
		app.ShowTD(thingID)
	case CmdShowStatus:
		thingID := getThingID()
		app.ShowStatus(thingID, false)
	case CmdSubscribe:
		thingID := getThingID()
		app.ShowSubscribe(thingID)

	default:
		fmt.Printf("\nUnknown command: %s\n", cmd)
	}
}

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
	"github.com/hiveot/hivekit/go/cells/router"
	"github.com/hiveot/hivekit/go/examples/cli/cliex"
	consumerrecipe "github.com/hiveot/hivekit/go/factory/recipes/consumer"
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

	env.RpcTimeout = time.Minute * 3 // for testing
	env.Print(env.Verbose)

	args := flag.Args()
	if len(args) == 0 {
		env.Print(0)
		flag.Usage()
		return
	}

	cmd := args[0]

	// helper when an arg is expected.
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

	// Start the CLI consumer recipe cells.
	// the env must have a clientID set and a named auth token or client cert.

	// FIXME: chain breaks if client cannot find a  serverURL
	r, f, err := consumerrecipe.NewConsumerRecipe(env, false)
	if err != nil {
		os.Exit(1)
	}

	// Set default credentials for connecting to devices using the router.
	// For demonstration purpose only. Normally you'd set per-device credentials.
	defaultAuthToken, _ := env.GetAuthToken()
	rtr := api.GetFactoryCell[router.IRouterService](f, router.RouterCellType)
	if defaultAuthToken != "" {
		rtr.AddCredentials("", env.ClientID, defaultAuthToken, td.SecSchemeBearer)
	}
	caCert, err := env.GetCACert()
	discoClient := r.GetDiscovery()
	dirClient := r.GetDirectory()
	app := cliex.NewCliex(appConfig, r.Consumer, discoClient, dirClient, caCert)

	f.Start()

	switch cmd {
	case CmdDiscover, "disco":
		app.ShowDiscovery()
	case CmdListDir:
		// optional directory thingID
		thingID := ""
		if len(args) > 1 {
			thingID = args[1]
		}
		app.ListDir(env, thingID)
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

package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/examples/example2/cliapp"
	"github.com/hiveot/hivekit/go/modules/directory"
	factorypkg "github.com/hiveot/hivekit/go/modules/factory/pkg"
	"github.com/hiveot/hivekit/go/modules/factory/recipes"
	"github.com/hiveot/hivekit/go/modules/transport/discovery"
	"github.com/hiveot/hivekit/go/utils"
)

// commands:
//	wotcli  [-txt] discover           discover devices on the network
//	wotcli  td  <thingID>             show the TD of a discovered thing
//	wotcli  status  <thingID>         show the current status of a thing
//	wotcli  subscribe  <thingID>      subscribe to updates of a thing

const (
	CmdDiscover   = "discover"
	CmdListDir    = "dir"
	CmdShowTD     = "td"
	CmdShowStatus = "status"
	CmdSubscribe  = "subscribe"
)

var appConfig cliapp.CliAppConfig

func main() {
	// var subscribe bool
	utils.SetLogging("warn", "")

	// environment defaults
	flag.BoolVar(&appConfig.Subscribe, "subscribe", appConfig.Subscribe, "Subscribe to events or property changes until ^C")
	flag.BoolVar(&appConfig.Verbose, "v", appConfig.Verbose, "Show more detailed output")
	flag.BoolVar(&appConfig.NoDisco, "nd", appConfig.NoDisco, "Do not start with discovery")

	env := api.NewAppEnvironment("", true)
	_ = env
	args := flag.Args()
	if len(args) == 0 {
		fmt.Printf("wotcli [options] command  \n\n")
		fmt.Println("Where command is one of:")
		fmt.Printf(" %-10s           Discover WoT devices and directories\n", CmdDiscover)
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

	f := factorypkg.NewModuleFactory(env, nil)
	r := recipes.NewConsumerRecipe(f)

	err := r.Start()
	if err != nil {
		os.Exit(1)
	}
	// not really needed as this is manually instantiated but this is an easy way.
	// appDef := &api.ModuleDefinition{
	// 	Type:        "CliApp",
	// 	Constructor: cliapp.NewCliAppFactory,
	// 	Config:      appConfig,
	// }
	// app, _ := cliapp.NewCliAppFactory(f, appDef)

	discoClient := api.GetFactoryModule[discovery.IDiscoveryClient](f, discovery.DiscoveryClientModuleType)
	dirClient := api.GetFactoryModule[directory.IDirectoryClient](f, directory.DirectoryClientModuleType)
	app := cliapp.NewCliApp(appConfig, discoClient, dirClient, f.GetEnvironment().CaCert)

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

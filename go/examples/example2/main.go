package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/hiveot/hivekit/go/examples/example2/wotcli"
	"github.com/hiveot/hivekit/go/examples/wotco"
	"github.com/hiveot/hivekit/go/modules/factory"
	routerpkg "github.com/hiveot/hivekit/go/modules/router/pkg"
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

func main() {
	var subscribe bool
	var verbose bool
	utils.SetLogging("warn", "")

	// environment defaults
	flag.BoolVar(&subscribe, "subscribe", subscribe, "Subscribe to events or property changes until ^C")
	flag.BoolVar(&verbose, "v", verbose, "Show more detailed output")

	env := factory.NewAppEnvironment("", true)
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

	// Ignore the certificate check just for this example. Dont do this.
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	co := wotco.NewWotConsumer()
	co.SetTimeout(time.Minute)
	// run the router without CA. Don't try this at home.
	// the WotConsumer has a list of collected TDs for use by the router
	r := routerpkg.NewRouterService("", co.GetTD, nil, nil)
	err := r.Start()
	if err != nil {
		slog.Error(err.Error())
	}
	co.SetRequestSink(r.HandleRequest)
	r.SetNotificationSink(co.HandleNotification)

	// discover.Discover(filterType, filterAddr, showAff, showTXT, showTD, waitTime)
	switch cmd {
	case CmdDiscover:
		wotcli.ShowDiscovery(co, verbose)
	case CmdListDir:
		wotcli.ListDir(co)
	case CmdShowTD:
		thingID := getArgs()
		wotcli.ShowTD(co, thingID)
	case CmdShowStatus:
		thingID := getArgs()
		wotcli.ShowStatus(co, thingID, subscribe)
	default:
		fmt.Printf("\nUnknown command: %s\n", cmd)
	}
}

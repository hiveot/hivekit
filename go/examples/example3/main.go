package main

import (
	"crypto/tls"
	"flag"
	"net/http"

	"github.com/hiveot/hivekit/go/examples/example3/wotcli"
	"github.com/hiveot/hivekit/go/modules/factory"
	factorypkg "github.com/hiveot/hivekit/go/modules/factory/pkg"
	"github.com/hiveot/hivekit/go/modules/transports"
	httptransportpkg "github.com/hiveot/hivekit/go/modules/transports/httptransport/pkg"
	"github.com/hiveot/hivekit/go/modules/transports/wss"
	wsspkg "github.com/hiveot/hivekit/go/modules/transports/wss/pkg"
	"github.com/hiveot/hivekit/go/utils"
)

// Recipe for a CLI that provides ability to discovery and query a device
var recipe = factorypkg.FactoryRecipe{
	ModuleDefs: map[string]factory.ModuleDefinition{
		// todo: this needs a CA
		transports.HttpServerModuleType: {
			Constructor: httptransportpkg.NewHttpTransportServerFactory,
		},
		wss.HiveotWebsocketClientModuleType: {
			Constructor: wsspkg.NewHiveotWssClientFactory,
		},
		wotcli.WotCLIModuleType: {
			Constructor: wotcli.NewWotCLIFactory,
		},
	},

	// This chain defines the complete application
	ModuleChain: []string{
		wotcli.WotCLIModuleType,
		wss.HiveotWebsocketClientModuleType,
	},
}

func main() {
	utils.SetLogging("warn", "")
	var showTD bool
	var showTXT bool
	var filterAddr string
	var waitTime int = 3
	var filterType string

	flag.StringVar(&filterAddr, "addr", filterAddr, "Filter on a specific address")
	flag.StringVar(&filterType, "type", filterType, "Filter on type 'directory' or 'thing'")
	flag.BoolVar(&showTD, "td", showTD, "Show the discovered TD")
	flag.BoolVar(&showTXT, "txt", showTXT, "Show the DNS-SD TXT record entries")
	flag.IntVar(&waitTime, "wait", waitTime, "Nr of seconds to wait for the result")
	flag.Parse()

	// Ignore the certificate check just for this example. Dont do this.
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	// // start the factory using the default installation home directory
	// env := factory.NewAppEnvironment("~/bin/hiveot", true)
	// f := factorypkg.NewModuleFactory(env, nil)
	// err := f.Start()
	// if err == nil {
	// 	// run it with the recipe
	// 	r := factorypkg.NewFactoryRecipe(recipe.ModuleDefs, recipe.ModuleChain)
	// 	_ = r
	// 	r.Start(f)
	// }
	// // h
	// f.Stop()
}

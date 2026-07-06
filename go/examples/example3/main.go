package main

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/examples/example3/tuiapp"
	factorypkg "github.com/hiveot/hivekit/go/modules/factory/pkg"
	"github.com/hiveot/hivekit/go/modules/factory/recipes"
	"github.com/hiveot/hivekit/go/utils"
)

func main() {
	// utils.SetLogging("warn", "")
	// log to file to avoid messing up the tui
	utils.SetLogging("info", "/tmp/example3.log")

	// run the EFR trio: env, factory and recipe
	env := api.NewAppEnvironment("", true)
	f := factorypkg.NewModuleFactory(env, nil)
	r := recipes.NewConsumerRecipe(f)

	// Ignore the certificate check just for this example. Dont do this at home.
	// http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	// co := wotco.NewWotConsumer(nil, time.Minute)
	// co := consumer.NewConsumer()
	// co.SetTimeout(time.Minute)
	// run the router without CA. Don't try this at home.
	// r := routerpkg.NewRouterService("", co.GetTD, nil, nil, time.Minute)
	// co.SetRequestSink(r)
	// r.SetNotificationSink(co)
	err := r.Start()
	app := tuiapp.NewTuiApp(f)
	app.Run()
	if err != nil {
		println("Tui failed to start: ", err.Error())
	} else {
		println("Done\n")
	}
}

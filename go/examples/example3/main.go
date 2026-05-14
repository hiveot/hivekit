package main

import (
	"crypto/tls"
	"net/http"
	"time"

	"github.com/hiveot/hivekit/go/examples/example3/wottui"
	"github.com/hiveot/hivekit/go/examples/wotco"
	routerpkg "github.com/hiveot/hivekit/go/modules/router/pkg"
	"github.com/hiveot/hivekit/go/utils"
)

func main() {
	// utils.SetLogging("warn", "")
	// log to file to avoid messing up the tui
	utils.SetLogging("info", "/tmp/example3.log")

	// Ignore the certificate check just for this example. Dont do this at home.
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	co := wotco.NewWotConsumer()
	co.SetTimeout(time.Minute)
	// run the router without CA. Don't try this at home.
	r := routerpkg.NewRouterService("", co.GetTD, nil, nil)
	co.SetRequestSink(r.HandleRequest)
	r.SetNotificationSink(co.HandleNotification)

	app := wottui.NewTuiApp(co)
	app.Run()
	println("Done\n")
}

package main

import (
	"crypto/tls"
	"net/http"
	"time"

	"github.com/hiveot/hivekit/go/examples/example3/wottui"
	"github.com/hiveot/hivekit/go/examples/wotmodel"
	"github.com/hiveot/hivekit/go/utils"
)

func main() {
	// utils.SetLogging("warn", "")
	// log to file to avoid messing up the tui
	utils.SetLogging("info", "/tmp/example3.log")

	// Ignore the certificate check just for this example. Dont do this at home.
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	model := wotmodel.NewWotModel()
	model.SetTimeout(time.Minute)

	app := wottui.NewAppView(model)
	app.Run()
	println("Done\n")
}

package main

import (
	"crypto/tls"
	"flag"
	"net/http"

	"github.com/hiveot/hivekit/go/examples/example2/discover"
	"github.com/hiveot/hivekit/go/utils"
)

func main() {
	utils.SetLogging("warn", "")
	var showAff bool
	var showTD bool
	var showTXT bool
	var filterAddr string
	var waitTime int = 3
	var filterType string

	flag.StringVar(&filterAddr, "addr", filterAddr, "Filter on a specific address")
	flag.StringVar(&filterType, "type", filterType, "Filter on type 'directory' or 'thing'")
	flag.BoolVar(&showAff, "aff", showAff, "Show the TD affordances")
	flag.BoolVar(&showTD, "td", showTD, "Show the discovered TD")
	flag.BoolVar(&showTXT, "txt", showTXT, "Show the DNS-SD TXT record entries")
	flag.IntVar(&waitTime, "wait", waitTime, "Nr of seconds to wait for the result")
	flag.Parse()

	// Ignore the certificate check just for this example. Dont do this.
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	discover.Discover(filterType, filterAddr, showAff, showTXT, showTD, waitTime)
}

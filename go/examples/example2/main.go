package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	discoverypkg "github.com/hiveot/hivekit/go/modules/transports/discovery/pkg"
	"github.com/hiveot/hivekit/go/utils"
	jsoniter "github.com/json-iterator/go"
)

// discover things and directories on the network
func Discover(filterType string, filterAddr string, showTXT bool, showTD bool, waitSec int) {
	nrFound := 0
	filterType = strings.ToLower(filterType)
	filterAddr = strings.ToLower(filterAddr)

	disco := discoverypkg.NewDiscoveryClient()
	fmt.Println("Discovered Things and Directories on the local network")
	fmt.Printf("Type       Address    Port   Instance             Schema   TD URL   \n")
	fmt.Printf("---------- ---------- -----  -------------------  -------  -------  \n")
	waitDuration := time.Duration(waitSec) * time.Second

	disco.DiscoverThings("", waitDuration, func(r *discoverypkg.DiscoveryResult) bool {
		if filterType != "" && filterType != strings.ToLower(r.Type) {
			return false
		}
		if filterAddr != "" && filterAddr != strings.ToLower(r.Addr) {
			return false
		}

		tdURL := r.AsURL()
		fmt.Printf("%-10s %-10s %-5d  %-20s %-8s %s \n",
			r.Type, r.Addr, r.Port, r.Instance, r.Schema, tdURL)
		if showTXT {
			for k, v := range r.Params {
				fmt.Printf("  %10s: %s\n", k, v)
			}
		}
		if showTD && tdURL != "" {
			resp, err := http.Get(tdURL)
			var rawObj any
			var pretty []byte
			if err == nil {
				raw, _ := io.ReadAll(resp.Body)
				err = jsoniter.Unmarshal(raw, &rawObj)
			}
			if err == nil {
				pretty, err = jsoniter.MarshalIndent(rawObj, "", "  ")
				fmt.Println(string(pretty))
			}
			if err != nil {
				fmt.Printf(" Error reading TD: %s\n", err.Error())
			}
		}
		fmt.Println()
		nrFound++
		return false
	})
	fmt.Printf("Found %d records\n", nrFound)
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

	Discover(filterType, filterAddr, showTXT, showTD, waitTime)
}

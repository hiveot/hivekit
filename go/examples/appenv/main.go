package main

import (
	"fmt"
	"os"
	"time"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/utils"
)

// Commandline utility for displaying the application environment

// TODO

func main() {

	// accept commandline flags
	appenv := api.NewHiveEnvironment("", true)

	// for now just print the location
	fmt.Printf("Application Environment\n")
	fmt.Printf(" This binary:      %s\n", os.Args[0])

	fmt.Printf(" home directory:   %s\n", appenv.HomeDir)
	fmt.Printf(" bin directory:    %s\n", appenv.BinDir)
	fmt.Printf(" certs directory:  %s\n", appenv.CertsDir)
	fmt.Printf(" config directory: %s\n", appenv.ConfigDir)
	fmt.Printf(" logs directory:   %s\n", appenv.LogsDir)
	fmt.Printf(" stores directory: %s\n", appenv.StoresDir)

	// see if the CA can be loaded
	caCert, err := appenv.GetCACert()
	if err != nil {
		fmt.Printf(" CA cert cannot be loaded: %s\n", err.Error())
	} else {
		fmt.Printf(" CA cert was successfully loaded\n")
		fmt.Printf(" - CN: %s\n", caCert.Subject.CommonName)
		fmt.Printf(" - OU: %s\n", caCert.Subject.OrganizationalUnit)
		fmt.Printf(" - Country: %s\n", caCert.Subject.Country)
		fmt.Printf(" - Province: %s\n", caCert.Subject.Province)
		fmt.Printf(" - Locality: %s\n", caCert.Subject.Locality)
		fmt.Printf(" - Organization: %s\n", caCert.Subject.Organization)
		fmt.Printf(" - valid until: %s\n", caCert.NotAfter)
	}
	// same for client cert
	clientCert, err := appenv.GetClientCert()
	if err != nil {
		fmt.Printf(" Client cert not loaded: %s\n", err.Error())
	} else {
		chain, _ := utils.TLSCertToX509(clientCert)
		cert := chain[0]
		fmt.Printf(" Client Cert cert was successfully loaded\n")
		fmt.Printf(" - CN: %s\n", cert.Subject.CommonName)
		fmt.Printf(" - OU: %s\n", cert.Subject.OrganizationalUnit)
		fmt.Printf(" - valid until: %s\n", cert.NotAfter)
	}
	fmt.Printf(" clientID:         %s\n", appenv.ClientID)
	fmt.Printf(" serverURL:        %s\n", appenv.ServerURL)
	fmt.Printf(" rpcTimeout:       %d msec\n", appenv.RpcTimeout/time.Millisecond)

}

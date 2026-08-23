package certs_service

import (
	"os"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/cells/certs"
	"github.com/hiveot/hivekit/go/cells/certs/internal"
)

// NewInitFactoryCerts if a factory initialization cell to ensure certificates needed
// to run the servers or administrator exist.
//
// Intended to ensure all certs are in place before running the services.
// This can be used together with the certs service.
//
// If certificates are created they are valid for the default 90 days:
// 1. Create a self-signed CA Key if it doesn't exist.
// 2. Create a self-signed CA Certificate if it doesn't exist.
// 3. Create a self-signed admin client TLS certificate if it doesn't exist.
// 4. Create a self-signed server TLS certificate if it doesn't exist. (in memory only)
//
// This returns nil so it won't be included in request handling, just does some setup
// at startup.
func NewInitFactoryCerts(
	f api.ICellFactory, md *api.CellDefinition) (api.IHiveCell, error) {

	var err error

	// Update the environment with the CA, server and admin client certificate
	env := f.GetEnvironment()

	// Load or create a self-signed CA
	caCert, caPrivKey := internal.LoadCACert(env.CertsDir)
	if caCert == nil || caPrivKey == nil {
		caCert, caPrivKey, err = internal.CreateSelfSignedCACert(
			env.CertsDir, caPrivKey, certs.DefaultCAValidityPeriod)
	}
	if err != nil {
		return nil, err
	}

	// Load or create a self-signed server cert
	env.SetCACert(caCert)

	cfg := &certs.CertsConfig{
		CertsDir: env.CertsDir,
		CaCert:   caCert,
		CaKey:    caPrivKey,
	}
	serverCert, err := env.GetServerCert()
	if err != nil {
		// default server cert not found, create it
		hostname, _ := os.Hostname()
		serverCert, err = internal.CreateSelfSignedServerCert(
			api.DefaultServerName, hostname, cfg, certs.DefaultServerValidityPeriod)
		env.SetServerCert(serverCert)
	}
	clientCert, err := env.GetClientCert()
	if err != nil {
		clientCert, err = internal.CreateSelfSignedClientCert(
			api.DefaultAdminUserID, cfg, "ou", certs.DefaultAdminValidityPeriod)
		env.SetClientCert(clientCert)
	}
	// the job here is done. No need to return an instance as it doesnt handle requests
	// or notifications.
	return nil, nil
}

package certs_service

import (
	"crypto"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/modules/certs"
	"github.com/hiveot/hivekit/go/modules/certs/internal"
)

// NewInitFactoryCerts if a factory initialization method to ensure it has
// certificates needed to run the servers or administrator.
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
// This returns nil so it won't be added to the module chain, just does some setup
// at startup.
func NewInitFactoryCerts(
	f api.IModuleFactory, md *api.ModuleDefinition) (api.IHiveModule, error) {

	var err error
	var caPrivKey crypto.Signer

	// Update the environment with the CA, server and admin client certificate
	env := f.GetEnvironment()

	env.CaCert, caPrivKey, err = internal.LoadOrCreateSelfSignedCACert(
		env.CertsDir, certs.DefaultCAValidityPeriod)
	if err != nil {
		return nil, err
	}

	cfg := &certs.CertsConfig{
		CertsDir: env.CertsDir,
		CaCert:   env.CaCert,
		CaKey:    caPrivKey,
	}

	env.ServerCert, err = internal.LoadOrCreateSelfSignedServerCert(
		api.DefaultServerName, cfg, certs.DefaultServerValidityPeriod)

	env.ClientCert, err = internal.LoadOrCreateSelfSignedClientCert(
		certs.DefaultAdminID, cfg, "ou", certs.DefaultAdminValidityPeriod)

	// the job here is done. No need to return a module
	return nil, nil
}

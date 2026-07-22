package certs_service

import (
	"crypto"
	"os"
	"path"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/modules/certs/internal/providers/selfsigned"
	"github.com/hiveot/hivekit/go/utils"
)

// NewInitFactoryCerts if a factory initialization method to ensure it has
// certificates needed to run the servers.
//
// If a CA and server certs are already loaded then this does nothing.
// If a CA with key is loaded but the server cert is not then create a server cert.
// If no certs are available then generate a self-signed CA with server cert.
//
// Any certs created are in-memory only. No files on disk are changed as certificate
// management is a separate concern.
//
// This returns nil so it won't be added to the module chain, just does some setup
// at startup.
func NewInitFactoryCerts(f api.IModuleFactory, md *api.ModuleDefinition) (api.IHiveModule, error) {
	var err error
	var caPrivKey crypto.PrivateKey
	var caPubKey crypto.PublicKey

	env := f.GetEnvironment()

	// if certs are in place there is nothing to do
	if env.CaCert != nil && env.TLSCert != nil {
		// return nil is not an error are this module's work is done
		return nil, nil
	}

	// to ensure a CA and server cert can be created, a CA private key is required
	caKeyPath := path.Join(env.CertsDir, api.DefaultCaKeyFile)
	caPrivKey, caPubKey, err = utils.LoadPrivateKey(caKeyPath)
	if err != nil {
		// no luck, need a new set of keys, all certs need to be created
		caPrivKey, caPubKey = utils.NewKey(utils.KeyTypeECDSA)
		// even if there was a CA, without a key it cannot be used to create a server cert
		env.CaCert = nil
	}

	// if no CA exists then create it
	if env.CaCert == nil {
		env.CaCert, err = selfsigned.CreateCAFromKey(
			"CA", "BC", "local", "HiveOT", env.AppID, 365, caPrivKey, caPubKey)
	}

	// and include a new server cert with hostname and outbound ip
	names := []string{}
	hostname, err := os.Hostname()
	if err == nil {
		names = append(names, hostname)
	}
	ip := utils.GetOutboundIP("")
	names = append(names, ip.String())

	serverPrivKey, serverPubKey := utils.NewKey(utils.KeyTypeECDSA)
	serverX509, err := selfsigned.CreateSelfSignedServerCert(
		hostname, "HiveOT", 365, serverPubKey, names, env.CaCert, caPrivKey)
	env.TLSCert = utils.X509CertToTLS(serverX509, serverPrivKey)

	// the job here is done. No need to return a module
	return nil, nil
}

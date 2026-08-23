package internal

import (
	"crypto"
	"crypto/tls"
	"crypto/x509"
	"log/slog"
	"path"
	"path/filepath"
	"time"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/cells/certs"
	"github.com/hiveot/hivekit/go/utils"
)

// Create a HiveOT self-signed server certificate and keys for the current machine.
//
// This adds os.Hostname, the outbound IP and the given address to the certificate
// SAN names.
//
//	serverName common name under which to save the certificate
//	address is the IP or domain name to include as the SAN names.
//	certsDir is the location of certificates
//	validity is the CA's validity duration
//
// This returns the server TLS certificate, or an error if none can be created or stored.
func CreateSelfSignedServerCert(
	serverName string, address string,
	cfg *certs.CertsConfig, validity time.Duration) (*tls.Certificate, error) {

	// and include a new server cert with hostname and outbound ip
	names := []string{}
	ip := utils.GetOutboundIP("")
	if ip != nil {
		names = append(names, ip.String())
	}
	if address != "" {
		names = append(names, address)
	}

	serverPrivKey, serverPubKey := utils.NewKey(utils.KeyTypeECDSA)
	serverX509, err := utils.CreateServerCert(
		serverName, "HiveOT", cfg.Country, cfg.Province, cfg.Locality, cfg.Org,
		names, validity, serverPubKey, cfg.CaCert, cfg.CaKey)

	if err != nil {
		return nil, err
	}
	tlsCert := utils.X509CertToTLS(serverX509, serverPrivKey)
	return tlsCert, nil
}

// Create and save a HiveOT self-signed client certificate.
//
// If a private key is found it is re-used.
// If no private key is provided then one is generated using ED25519.
//
//	clientID under which the certificate is saved
//	certsDir is the location of certificates
//	ou for creation
//	validity for creation
//
// This returns the CA with private key, or an error if none can be created or stored.
func CreateSelfSignedClientCert(
	clientID string, cfg *certs.CertsConfig, ou string, validity time.Duration) (
	cert *tls.Certificate, err error) {

	certPath := filepath.Join(cfg.CertsDir, clientID+api.DefaultCertFileSuffix)
	keyPath := filepath.Join(cfg.CertsDir, clientID+api.DefaultPrivKeyFileSuffix)

	// create the cert and key
	privKey, _ := utils.NewEd25519Key()
	x509Cert, err := utils.CreateClientCert(
		clientID, ou,
		cfg.Country, cfg.Province, cfg.Locality, cfg.Org,
		validity, privKey.Public(), cfg.CaCert, cfg.CaKey)

	tlsCert := utils.X509CertToTLS(x509Cert, privKey)

	err = utils.SaveTLSCert(tlsCert, certPath, keyPath)

	return tlsCert, err
}

// Create/save a HiveOT self-signed CA certificate using the given key.
// Intended for local and client cert use.
//
// If the privateKey is provided then the new CA certificate will be still be
// able to match old certificates as long as the issuer matches.
//
// If no private key is provided then one is generated using ED25519.
//
//	certsDir is the location of certificates
//	caKey the private key used to sign the CA selfsigned certificate. nil to generate.
//	validity is the CA's validity duration
//
// This returns the CA with private key, or an error if none can be created or stored.
func CreateSelfSignedCACert(
	certsDir string, caSignerKey crypto.Signer, validity time.Duration) (
	caCert *x509.Certificate, caKey crypto.Signer, err error) {

	var caPubKey crypto.PublicKey

	caKeyPath := path.Join(certsDir, api.DefaultCaKeyFile)
	caCertPath := path.Join(certsDir, api.DefaultCaCertFile)

	// load or create the self-signed CA private key
	if caSignerKey == nil {
		caSignerKey, caPubKey = utils.NewEd25519Key()

		slog.Warn("CreateSelfSignedCACert: New CA key", "caKeyPath", caKeyPath)
		err = utils.SavePrivateKey(caSignerKey, caKeyPath)
	}

	// note that if a private key exists but the CA is recreated the new
	// CA will be valid with old cert IF the issuer information matches.
	caCert, err = utils.CreateCACert(
		"hiveot", "CA", "BC", "HiveOT zone", "HiveOT",
		validity, caSignerKey, caPubKey)

	if err == nil {
		slog.Warn("CreateSelfSignedCACert: New CA at", "caCertPath", caCertPath)
		err = utils.SaveX509Cert(caCert, caCertPath)
	}

	if err != nil {
		return nil, nil, err
	}

	return caCert, caSignerKey, err
}

// Load the default CA certificate
// This returns the CA certificate and optional private key if found.
// This returns nil for the CA or key if not found.
func LoadCACert(certsDir string) (*x509.Certificate, crypto.Signer) {

	caKeyPath := path.Join(certsDir, api.DefaultCaKeyFile)
	caKey, caPub, _ := utils.LoadPrivateKey(caKeyPath)
	_ = caPub

	caCertPath := path.Join(certsDir, api.DefaultCaCertFile)
	caCert, _ := utils.LoadCACert(caCertPath)
	return caCert, caKey
}

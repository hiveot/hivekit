package internal

import (
	"crypto"
	"crypto/tls"
	"crypto/x509"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/modules/certs"
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

// Load or create/save a HiveOT self-signed client certificate.
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
func LoadOrCreateSelfSignedClientCert(
	clientID string, cfg *certs.CertsConfig, ou string, validity time.Duration) (
	cert *tls.Certificate, err error) {

	certPath := filepath.Join(cfg.CertsDir, clientID+api.DefaultCertFileSuffix)
	keyPath := filepath.Join(cfg.CertsDir, clientID+api.DefaultKeyFileSuffix)
	tlsCert, err := utils.LoadTLSCert(certPath, keyPath)
	if err == nil {
		return tlsCert, err
	}
	// create the cert and key
	privKey, _ := utils.NewEd25519Key()

	x509Cert, err := utils.CreateClientCert(
		clientID, ou,
		cfg.Country, cfg.Province, cfg.Locality, cfg.Org,
		validity, privKey.Public(), cfg.CaCert, cfg.CaKey)

	tlsCert = utils.X509CertToTLS(x509Cert, privKey)

	err = utils.SaveTLSCert(tlsCert, certPath, keyPath)

	return tlsCert, err
}

// Load or create/save a HiveOT self-signed server certificate and keys for
// the current machine.
//
// This adds os.Hostname and outbound IP to the certificate names.
// If a private key is found it is re-used.
// If no private key is provided then one is generated using ED25519.
//
//	serverName under which the certificate is saved
//	certsDir is the location of certificates
//	validity is the CA's validity duration
//
// This returns the server TLS certificate, or an error if none can be created or stored.
func LoadOrCreateSelfSignedServerCert(
	serverName string, cfg *certs.CertsConfig, validity time.Duration) (*tls.Certificate, error) {

	certPath := path.Join(cfg.CertsDir, serverName+api.DefaultCertFileSuffix)
	keyPath := path.Join(cfg.CertsDir, serverName+api.DefaultKeyFileSuffix)
	tlsCert, err := utils.LoadTLSCert(certPath, keyPath)
	if err == nil {
		return tlsCert, err
	}

	hostname, err := os.Hostname()
	tlsCert, err = CreateSelfSignedServerCert(serverName, hostname, cfg, validity)
	err = utils.SaveTLSCert(tlsCert, certPath, keyPath)

	return tlsCert, err
}

// Load or create/save a HiveOT self-signed CA certificate and keys.
// Intended for local and client cert use.
//
// If a private key is found it is re-used.
// If no private key is provided then one is generated using ED25519.
//
//	certsDir is the location of certificates
//	validity is the CA's validity duration
//
// This returns the CA with private key, or an error if none can be created or stored.
func LoadOrCreateSelfSignedCACert(certsDir string, validity time.Duration) (
	caCert *x509.Certificate, caPrivKey crypto.Signer, err error) {

	var caPubKey crypto.PublicKey

	caKeyPath := path.Join(certsDir, api.DefaultCaKeyFile)
	caCertPath := path.Join(certsDir, api.DefaultCaCertFile)

	// load or create the self-signed CA private key
	caPrivKey, caPubKey, err = utils.LoadPrivateKey(caKeyPath)
	if err != nil {
		caPrivKey, caPubKey = utils.NewEd25519Key()

		slog.Warn("Created new CA private key", "caKeyPath", caKeyPath)
		err = utils.SavePrivateKey(caPrivKey, caKeyPath)
	}

	// load or create the CA cert
	caCert, err = utils.LoadCACert(caCertPath)
	if err != nil {
		// note that if a private key exists but the CA is recreated the new
		// CA will be valid with old cert IF the issuer information matches.
		caCert, err = utils.CreateCACert(
			"hiveot", "CA", "BC", "HiveOT zone", "HiveOT",
			validity, caPrivKey, caPubKey)

		if err == nil {
			slog.Warn("Created self-signed CA", "caCertPath", caCertPath)
			err = utils.SaveX509Cert(caCert, caCertPath)
		}
	}
	if err != nil {
		return nil, nil, err
	}

	return caCert, caPrivKey, err
}

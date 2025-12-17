package service

import (
	"crypto/x509"
	"log/slog"
	"path"

	"github.com/hiveot/hivekit/go/modules/services/certs"
	"github.com/hiveot/hivekit/go/modules/services/certs/keys"
	"github.com/hiveot/hivekit/go/utils/net"
)

// Defaults for a self-signed CA
const DefaultCA_CN = "HiveOT"
const DefaultCA_Country = "Earth"
const DefaultCA_Locality = "HiveOT zone"
const DefaultCA_Org = "Internet of things"
const DefaultCA_Province = "One World"
const DefaultCA_Validity = 365*20 + 5

// Certificate management service for managing certificates.
// This implements the certs.ICerts interface.
type CertsService struct {
	// location of certificates or "" for in-memory only
	certsDir string

	// ca certificate or nil if none found
	caCert *x509.Certificate
	// ca key-pair
	caKey keys.IHiveKey
}

// Create and save a HiveOT self-signed CA certificate and keys.
//
// If a directory is configured, save the CA in the directory.
// If the directory already contains a CA then do nothing and return an error.
// If the directory contains a key-pair then use it instead of creating a new one.
// This uses ECDSA keys as browsers don't support ED25519 (2024)
//
// validityDays is the CA's validity in days
// This returns with the existing CA if available.
func (svc *CertsService) CreateCACert() (
	caCert *x509.Certificate, keyPair *keys.EcdsaKey, err error) {

	caCert, keyPair, err = CreateCA(
		DefaultCA_Country,
		DefaultCA_Province,
		DefaultCA_Locality,
		DefaultCA_Org,
		DefaultCA_CN,
		DefaultCA_Validity)
	return caCert, keyPair, err
}

// Create a TLS server certificate for the module with the given ID.
// localhost and 127.0.0.1 are always added to the SAN names.
//
// hostname will be added to the certificate SAN. If omitted, the outbound IP will be used.
// privKeyPem is the server's private ecdsa key in PEM format.
//
// The certificate will be signed by the CA on file, if present.
// If LetsEncrypt is configured then an internet connection is required. (a future feature)
func (svc *CertsService) CreateServerCert(moduleID string, hostname string, serverKey keys.IHiveKey) (*x509.Certificate, error) {

	// names are the SAN names to include with the certificate, localhost and 127.0.0.1 are always added
	names := []string{}
	if hostname != "" {
		names = append(names, hostname)
	} else {
		ip := net.GetOutboundIP("")
		names = append(names, ip.String())
	}
	serverCert, err := CreateServerCert(moduleID, DefaultCA_Org, 365, serverKey, names, svc.caCert, svc.caKey)

	// tlsCert := X509CertToTLS(serverCert, cfg.ServerKey)

	return serverCert, err
}

// Return the configured CA certificate
func (svc *CertsService) GetCACert() *x509.Certificate {
	return svc.caCert
}

// func (svc *CertsService) GetTLSCert(name string) *tls.Certificate {

// }

// Start a certificate management service.
// If a certificate directory is configured then attempt to load the CA keys and certificate.
// load keys and CA certificate from file if available
func (svc *CertsService) Start() (err error) {

	// Load the CA, certs and keys
	if svc.caCert == nil && svc.certsDir != "" {

	}

	// only load the ca key if the cert was loaded
	if svc.caCert != nil && svc.caKey == nil {
		caCertPath := path.Join(svc.certsDir, DefaultCaCertFile)
		slog.Info("loading CA certificate and key", "path", caCertPath)

		caKeyPath := path.Join(svc.certsDir, DefaultCaKeyFile)

		svc.caKey, err = keys.NewKeyFromFile(caKeyPath)
		_ = err

		svc.caCert, err = LoadX509CertFromPEM(caCertPath)
		if err != nil {
			// On first start there might not be a CA. Not a fatal error.
			slog.Warn("no CA certificate found", "path", caCertPath)
		}
	}

	return nil
}

// Stop the certificate management service
func (svc *CertsService) Stop() {
}

// Create a new instance of the certificate service
func NewCertsService(certsDir string) *CertsService {
	service := &CertsService{
		certsDir: certsDir,
	}
	var _ certs.ICertsService = service // interface check
	return service
}

package module

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"path"

	"github.com/hiveot/hivekit/go/modules/certs/keys"
	"github.com/hiveot/hivekit/go/modules/certs/module/selfsigned"
	"github.com/hiveot/hivekit/go/utils"
)

// Defaults for a self-signed CA
const DefaultCA_CN = "HiveOT"
const DefaultCA_Country = "Earth"
const DefaultCA_Locality = "HiveOT"
const DefaultCA_Org = "Internet of things"
const DefaultCA_Province = "One World"
const DefaultCA_Validity = 365*20 + 5

// Create and save a HiveOT self-signed CA certificate and keys.
//
// If a directory is configured, save the CA in the directory.
// If the directory already contains a CA then do nothing and return an error.
// If the directory contains a key-pair then use it instead of creating a new one.
// If no key-pair is provided this uses ECDSA keys, as browsers don't support ED25519 (2024)
//
// validityDays is the CA's validity in days
// This returns the CA, key or an error
func (m *CertsModule) CreateCACert() (
	caCert *x509.Certificate, keyPair keys.IHiveKey, err error) {

	caCert, keyPair, err = selfsigned.CreateCA(
		DefaultCA_Country,
		DefaultCA_Province,
		DefaultCA_Locality,
		DefaultCA_Org,
		DefaultCA_CN,
		DefaultCA_Validity)

	if m.certsDir != "" {
		// save the CA, but only if it won't overwrite an existing certificate
		caCertPath := path.Join(m.certsDir, DefaultCaCertFile)
		caKeyPath := path.Join(m.certsDir, DefaultCaKeyFile)

		if _, err := os.Stat(caCertPath); err == nil {
			err = fmt.Errorf("the CA certificate exists at %s", caCertPath)
			return nil, nil, err
		}
		if err == nil {
			err = keyPair.ExportPrivateToFile(caKeyPath)
		}
		if err == nil {
			err = SaveX509CertToPEM(caCert, caCertPath)
		}
	}
	return caCert, keyPair, err
}

// Create and save a TLS server certificate for the module with the given ID.
// localhost, 127.0.0.1 and the given hostname are added to the SAN names.
// If the certificate exists it is replaced.
//
// moduleID is the name under which the certificate is saved.
// hostname will be added to the certificate SAN. If omitted, the outbound IP will be used.
// serverKey is the server's private ecdsa or "" to generate a ecdsa key-pair.
//
// The certificate will be signed by the CA on file, if present.
// If LetsEncrypt is configured then an internet connection is required. (a future feature)
func (m *CertsModule) CreateServerCert(
	moduleID string, hostname string, serverKeyPair keys.IHiveKey) (tlsCert *tls.Certificate, err error) {

	// names are the SAN names to include with the certificate, localhost and 127.0.0.1 are always added
	names := []string{}
	if hostname != "" {
		names = append(names, hostname)
	} else {
		ip := utils.GetOutboundIP("")
		names = append(names, ip.String())
	}
	if serverKeyPair == nil {
		serverKeyPair = keys.NewEcdsaKey()
	}
	// use self-signed CA until letsencrypt is supported
	serverCert, err := selfsigned.CreateServerCert(
		moduleID, DefaultCA_Org, 365, serverKeyPair, names,
		m.caCert, m.caKeyPair)
	if err != nil {
		return tlsCert, err
	}
	tlsCert = X509CertToTLS(serverCert, serverKeyPair.PrivateKey())

	// persist the certificate
	certPath := path.Join(m.certsDir, moduleID+"Cert.pem")
	keyPath := path.Join(m.certsDir, moduleID+"Key.pem")
	err = SaveTLSCertToPEM(tlsCert, certPath, keyPath)

	return tlsCert, err
}

// Return the configured CA certificate
func (m *CertsModule) GetCACert() (*x509.Certificate, error) {
	if m.caCert == nil {
		return nil, fmt.Errorf("service not initialized")
	}
	return m.caCert, nil
}

// GetServerCert resturn the default shared server certificate.
func (m *CertsModule) GetDefaultServerTlsCert() (cert *tls.Certificate, err error) {

	if m.defaultServerCert == nil {
		return cert, fmt.Errorf("the default server certificate is not loaded")
	}
	return m.defaultServerCert, nil
}

// GetServerCert loads a previously save module server certificate from the
// certificate directory.
// The file names used are {moduleID}Cert.pem and {moduleID}Key.pem
func (m *CertsModule) LoadServerCert(moduleID string) (
	serverCert *tls.Certificate, err error) {

	if m.certsDir == "" {
		return serverCert, fmt.Errorf("certificate directory is not configured")
	}
	serverCertPath := path.Join(m.certsDir, moduleID+"Cert.pem")
	serverKeyPath := path.Join(m.certsDir, moduleID+"Key.pem")
	serverCert, err = LoadTLSCertFromPEM(serverCertPath, serverKeyPath)

	return serverCert, err
}

func (m *CertsModule) VerifyCert(moduleID string, cert *x509.Certificate) (err error) {
	cn, err := selfsigned.VerifyCert(cert, m.caCert)
	if err == nil {
		if cn != moduleID {
			err = fmt.Errorf("expected cn to be '%s' but it is '%s' instead", moduleID, cn)
		}
	}
	return err
}

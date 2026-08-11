package internal

import (
	"crypto"
	"crypto/rand"
	"crypto/x509"
	"fmt"
	"log/slog"
	"os"
	"path"
	"time"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/certs"
	"github.com/hiveot/hivekit/go/utils"
)

// Implementation of the certificate management service
type CertsServiceImpl struct {
	*modules.HiveModuleBase

	// default values to use
	country  string
	locality string
	org      string
	ou       string
	province string

	// The pool of CA certificates, including the system certs
	caCertPool *x509.CertPool

	// ca certificate or nil if none found
	caCert *x509.Certificate
	// ca key-pair
	caPrivKey crypto.Signer
	// certificate storage directory
	certsDir string

	// optional provider like letsencrypt engine
	provider certs.ICertProvider

	// server certificate and key
	// serverCert *tls.Certificate
}

// Create and save a HiveOT self-signed CA certificate and keys.
// Intended for local and client cert use.
//
// If no private key is provided then one is generated using ED25519
//
//	validity is the CA's validity duration
//
// This returns the CA with private key, or an error
func (svc *CertsServiceImpl) CreateCACert(validity time.Duration) (
	caCert *x509.Certificate, caPrivKey crypto.Signer, err error) {

	var caPubKey crypto.PublicKey

	if caPrivKey == nil {
		caPrivKey, caPubKey = utils.NewEd25519Key()
	} else {
		caPubKey = caPrivKey.Public()
	}
	caCert, err = utils.CreateCACert(
		"HiveOT-CA",
		svc.country, svc.province, svc.locality, svc.org,
		validity,
		caPrivKey,
		caPubKey)

	if err != nil {
		return nil, nil, err
	}
	svc.caCert = caCert
	svc.caPrivKey = caPrivKey

	if svc.certsDir != "" {
		// save the CA, but only if it won't overwrite an existing certificate
		caCertPath := path.Join(svc.certsDir, api.DefaultCaCertFile)
		caKeyPath := path.Join(svc.certsDir, api.DefaultCaKeyFile)

		if _, err = os.Stat(caCertPath); err == nil {
			err = fmt.Errorf("the CA certificate exists at %s", caCertPath)
			return nil, nil, err
		}
		err = utils.SavePrivateKey(caPrivKey, caKeyPath)
		if err == nil {
			err = utils.SaveX509Cert(caCert, caCertPath)
		}
	}
	return caCert, caPrivKey, err
}

// Create a client TLS cert. This requires having the CA cert and key.
func (svc *CertsServiceImpl) CreateClientCert(
	clientID string, ou string, validity time.Duration, clientPubKey crypto.PublicKey) (
	x509Cert *x509.Certificate, err error) {

	// just a sinmple wrapper around the library
	clientCert, err := utils.CreateClientCert(
		clientID, ou,
		svc.country, svc.province, svc.locality, svc.org,
		validity, clientPubKey, svc.caCert, svc.caPrivKey)

	return clientCert, err
}

// Create a new server cert.
// If a provider is configured then ask the provider for a certificate.
func (svc *CertsServiceImpl) CreateServerCert(
	serverName string, hostname string, validity time.Duration,
	serverPubKey crypto.PublicKey) (serverCert []*x509.Certificate, err error) {

	if svc.provider != nil {
		// use the provider
		serverCert, err = svc.provider.CreateServerCert(serverName, hostname, validity, serverPubKey)

	} else if svc.caCert != nil && svc.caPrivKey != nil {
		// create a self signed cert
		names := []string{hostname}
		x509Cert, err2 := utils.CreateServerCert(
			serverName, svc.ou,
			svc.country, svc.province, svc.locality, svc.org,
			names, validity, serverPubKey, svc.caCert, svc.caPrivKey)
		err = err2
		if err == nil {
			serverCert = []*x509.Certificate{x509Cert}
			certPath := path.Join(svc.certsDir, serverName+api.DefaultCertFileSuffix)
			err = utils.SaveX509CertChain(serverCert, certPath)
		}
	}
	return serverCert, err
}

// GetCACert returns the x509 CA certificate.
// Returns and error if a CA is not initialized or can not be returned.
func (svc *CertsServiceImpl) GetCACert() (*x509.Certificate, error) {
	var err error

	if svc.caCert != nil {
		return svc.caCert, nil
	}
	if svc.certsDir != "" {
		caCertPath := path.Join(svc.certsDir, api.DefaultCaCertFile)
		caKeyPath := path.Join(svc.certsDir, api.DefaultCaKeyFile)
		svc.caCert, svc.caPrivKey, err = utils.LoadCA(caCertPath, caKeyPath)
	}
	return svc.caCert, err
}

// GetServerCert returns a previously created server certificate.
//
// Return a saved cert or use a provider.
func (svc *CertsServiceImpl) GetServerCert(serverName string) (
	serverCert []*x509.Certificate, err error) {

	// saved certs can be provider out-of-band so always check for it.
	if svc.certsDir != "" {
		serverCertPath := path.Join(svc.certsDir, serverName+"Cert.pem")
		serverCert, err = utils.LoadX509Cert(serverCertPath)
	}
	if serverCert == nil && svc.provider != nil {
		serverCert, err = svc.provider.GetServerCert(serverName)
	}
	return serverCert, err
}

// Refresh the self signed CA.
// If no CA private key is found then this is an out-of-band provided CA and wont be refreshed.
// An error is returned if refresh was attempted and failed.
func (svc *CertsServiceImpl) RefreshCA(minRemaining time.Duration) error {

	minValidUntil := time.Now().Add(minRemaining)
	if svc.caPrivKey == nil {
		return nil
	}

	if minRemaining < 0 || svc.caCert.NotAfter.Before(minValidUntil) {

		validityPeriod := svc.caCert.NotAfter.Sub(svc.caCert.NotBefore)

		template := *svc.caCert
		template.NotBefore = time.Now().Add(-time.Second)
		template.NotAfter = time.Now().Add(validityPeriod)

		certDerBytes, err := x509.CreateCertificate(
			rand.Reader, &template, &template, svc.caPrivKey.Public(), svc.caPrivKey)
		if err != nil {
			return err
		}
		newCA, err := x509.ParseCertificate(certDerBytes)
		if err != nil {
			return err
		}
		svc.caCert = newCA
		if err == nil {
			caCertPath := path.Join(svc.certsDir, api.DefaultCaCertFile)
			err = utils.SaveX509Cert(newCA, caCertPath)
		}
	}
	return nil
}

// Refresh the server certificate if needed.
// The certificate is updated when its remaining validity is below minRemaining.
func (svc *CertsServiceImpl) Refresh(
	serverName string, minRemaining time.Duration) (chain []*x509.Certificate, err error) {

	minValidUntil := time.Now().Add(minRemaining)
	// this can be the cached self-signed or provider cert
	chain, err = svc.GetServerCert(serverName)
	if err != nil {
		// no certificate found
		return nil, err
	}
	oldCert := chain[0]

	if minRemaining < 0 || oldCert.NotAfter.Before(minValidUntil) {
		var newCert *x509.Certificate

		// refresh is needed
		validityPeriod := oldCert.NotAfter.Sub(oldCert.NotBefore)
		if svc.provider != nil {
			chain, err = svc.provider.Refresh(serverName)
		} else {
			// self signed
			newCert, err = utils.CreateCertFromTemplate(
				oldCert, validityPeriod, svc.caCert, svc.caPrivKey)
			chain[0] = newCert
		}
	} else {
		// no refresh needed
	}
	return chain, err
}

// Start readies the certificate management service for use.
//
// This loads the stored CA or creates a self-signed if none is found
// This loads the default TLS server certificate for use by servers or create a new if one isnt found
func (svc *CertsServiceImpl) Start() (err error) {
	slog.Info("Start: Starting certs service")

	if svc.certsDir == "" {
		return fmt.Errorf("Start: Missing certificate directory in config")
	}

	caCertPath := path.Join(svc.certsDir, api.DefaultCaCertFile)
	caKeyPath := path.Join(svc.certsDir, api.DefaultCaKeyFile)
	svc.caCert, svc.caPrivKey, err = utils.LoadCA(caCertPath, caKeyPath)
	// create missing CA key and cert
	if svc.caCert == nil || svc.caPrivKey == nil {
		// Make a clean start with cert and key.
		_ = os.Remove(caCertPath)
		_ = os.Remove(caKeyPath)
		svc.caCert, svc.caPrivKey, err = svc.CreateCACert(time.Hour * 365 * 10)
	}
	// setup the CA certificate pool for verifying certificates
	svc.caCertPool, _ = x509.SystemCertPool()
	svc.caCertPool.AddCert(svc.caCert)
	return err
}

// Stop any running actions
func (svc *CertsServiceImpl) Stop() {
	slog.Info("Stop: Stopping certs service")
	// m.service.Stop()
}

// Verify certificate against the system CA pool.
// Intended for client auth.
func (svc *CertsServiceImpl) VerifyClientCert(clientID string, clientCert *x509.Certificate) (err error) {

	if clientCert == nil {
		err = fmt.Errorf("VerifyClientCert: Nil cert from '%s'", clientID)
		return err
	}
	cn := clientCert.Subject.CommonName
	if cn == "" {
		err = fmt.Errorf("VerifyClientCert: cert from '%s' has no CommonName", clientID)
		return err
	}
	if cn != clientID {
		err = fmt.Errorf("VerifyClientCert: certificate cn '%s' does not match senderID '%s'", cn, clientID)
		return err
	}

	opts := x509.VerifyOptions{
		Roots:     svc.caCertPool,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	_, err = clientCert.Verify(opts)

	return err
}

// Create a new self-signed certificate provider
func NewCertsServiceImpl(config *certs.CertsConfig) *CertsServiceImpl {
	thingID := certs.DefaultCertsServiceThingID

	svc := &CertsServiceImpl{
		HiveModuleBase: modules.NewHiveModuleBase(thingID, 0),
		certsDir:       config.CertsDir,

		// Defaults for a self-signed CA
		country:  "",
		province: "",
		locality: "HiveOT",
		org:      "Internet of things",
	}

	var _ certs.ICertsService = svc // api validation
	return svc
}

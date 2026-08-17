package internal

import (
	"crypto"
	"crypto/rand"
	"crypto/x509"
	"fmt"
	"log/slog"
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

	config *certs.CertsConfig

	// The pool of CA certificates, including the system certs
	caCertPool *x509.CertPool

	// optional provider like letsencrypt engine
	provider certs.ICertProvider
}

// Create a client TLS cert. This requires having the CA cert and key.
func (svc *CertsServiceImpl) CreateClientCert(
	clientID string, ou string, validity time.Duration, clientPubKey crypto.PublicKey) (
	x509Cert *x509.Certificate, err error) {

	cfg := svc.config
	// just a sinmple wrapper around the library
	clientCert, err := utils.CreateClientCert(
		clientID, ou,
		cfg.Country, cfg.Province, cfg.Locality, cfg.Org,
		validity, clientPubKey, cfg.CaCert, cfg.CaKey)

	return clientCert, err
}

// Create a new server cert.
// If a provider is configured then ask the provider for a certificate, otherwise
// create a self-signed certificate.
func (svc *CertsServiceImpl) CreateServerCert(
	serverName string, hostname string, validity time.Duration,
	serverPubKey crypto.PublicKey) (serverCert []*x509.Certificate, err error) {

	cfg := svc.config
	if svc.provider != nil {
		// use the provider
		names := []string{hostname}
		serverCert, err = svc.provider.CreateServerCert(serverName, names, validity, serverPubKey)

	} else if cfg.CaCert != nil && cfg.CaKey != nil {
		// create a self signed cert
		names := []string{hostname}
		x509Cert, err2 := utils.CreateServerCert(
			serverName, certs.DefaultServerOU,
			cfg.Country, cfg.Province, cfg.Locality, cfg.Org,
			names, validity, serverPubKey, cfg.CaCert, cfg.CaKey)
		err = err2
		if err == nil {
			serverCert = []*x509.Certificate{x509Cert}
			certPath := path.Join(cfg.CertsDir, serverName+api.DefaultCertFileSuffix)
			err = utils.SaveX509CertChain(serverCert, certPath)
		}
	}
	return serverCert, err
}

// GetCACert returns the x509 CA certificate.
func (svc *CertsServiceImpl) GetCACert() *x509.Certificate {
	return svc.config.CaCert
}

// GetServerCert returns a previously created server certificate.
//
// Return a saved cert or use a provider.
func (svc *CertsServiceImpl) GetServerCert(serverName string) (
	serverCert []*x509.Certificate, err error) {

	// saved certs can be provider out-of-band so always check for it.
	if svc.config.CertsDir != "" {
		serverCertPath := path.Join(svc.config.CertsDir, serverName+"Cert.pem")
		serverCert, err = utils.LoadX509Cert(serverCertPath)
	}
	if serverCert == nil && svc.provider != nil {
		serverCert, err = svc.provider.GetServerCert(serverName)
	}
	return serverCert, err
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
				oldCert, validityPeriod, svc.config.CaCert, svc.config.CaKey)
			chain[0] = newCert
		}
	} else {
		// no refresh needed
	}
	return chain, err
}

// Refresh the self signed CA.
// If no CA private key is found then this is an out-of-band provided CA and wont be refreshed.
// An error is returned if refresh was attempted and failed.
func (svc *CertsServiceImpl) RefreshCA(minRemaining time.Duration) error {

	minValidUntil := time.Now().Add(minRemaining)
	cfg := svc.config
	if cfg.CaKey == nil {
		return nil
	}

	if minRemaining < 0 || cfg.CaCert.NotAfter.Before(minValidUntil) {

		validityPeriod := cfg.CaCert.NotAfter.Sub(cfg.CaCert.NotBefore)

		template := *cfg.CaCert
		template.NotBefore = time.Now().Add(-time.Second)
		template.NotAfter = time.Now().Add(validityPeriod)

		certDerBytes, err := x509.CreateCertificate(
			rand.Reader, &template, &template, cfg.CaKey.Public(), cfg.CaKey)
		if err != nil {
			return err
		}
		newCA, err := x509.ParseCertificate(certDerBytes)
		if err != nil {
			return err
		}
		cfg.CaCert = newCA
		if err == nil {
			caCertPath := path.Join(cfg.CertsDir, api.DefaultCaCertFile)
			err = utils.SaveX509Cert(newCA, caCertPath)
		}
	}
	return nil
}

// Start readies the certificate management service for use.
//
// If a self-signed CA isn't configured, it will be created.
//
// This returns an error if a certsdir is not provided.
func (svc *CertsServiceImpl) Start() (err error) {
	slog.Info("Start: Starting certs service")

	cfg := svc.config
	if cfg.CertsDir == "" {
		return fmt.Errorf("Start: Missing certificate directory")
	}
	if cfg.CaCert == nil {
		cfg.CaCert, cfg.CaKey, err = LoadOrCreateSelfSignedCACert(
			cfg.CertsDir, certs.DefaultCAValidityPeriod)
	}

	// include the CA in the certificate pool for verifying certificates
	svc.caCertPool, _ = x509.SystemCertPool()
	svc.caCertPool.AddCert(svc.config.CaCert)

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
		config:         config,
	}

	var _ certs.ICertsService = svc // api validation
	return svc
}

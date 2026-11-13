package internal

import (
	"crypto"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
	"path"
	"path/filepath"
	"time"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/wot"
	"github.com/hiveot/hivekit/go/cells/certs"
	"github.com/hiveot/hivekit/go/cells/thing"
	"github.com/hiveot/hivekit/go/utils"
)

// Implementation of the certificate management service
type CertsServiceImpl struct {
	*thing.ExposedThing

	// app environment with certs
	env *api.AppEnvironment

	// cert creation config
	config *certs.CertsConfig

	// The pool of CA certificates, including the system certs
	// caCertPool *x509.CertPool

	// load/save location of certificates
	// certsDir string

	// optional server certificate provider
	provider certs.ICertProvider
}

// Create a self-signed client cert. This requires having the CA cert and public key.
// validity of 0 uses the default
func (svc *CertsServiceImpl) CreateClientCert(
	clientID string, role string, validity time.Duration, clientPubKey crypto.PublicKey) (
	x509Cert *x509.Certificate, err error) {

	if clientID == "" {
		return nil, fmt.Errorf("CreateClientCert: Missing clientID")
	}
	if role == "" {
		return nil, fmt.Errorf("CreateClientTLSCert: Missing role")
	}
	if validity == 0 {
		validity = certs.DefaultClientValidityPeriod
	}

	cfg := svc.config
	caCert, caKey, err := svc.env.GetCACert()
	// just a sinmple wrapper around the library
	clientCert, err := utils.CreateClientCert(
		clientID, role,
		cfg.Country, cfg.Province, cfg.Locality, cfg.Org,
		validity, clientPubKey, caCert, caKey)

	return clientCert, err
}

// Create and save a self-signed client cert. This requires having the CA cert and key.
func (svc *CertsServiceImpl) CreateClientTLSCert(
	clientID string, role string, validity time.Duration) (
	tlsCert *tls.Certificate, err error) {

	if clientID == "" {
		return nil, fmt.Errorf("CreateClientTLSCert: Missing clientID")
	}
	if role == "" {
		return nil, fmt.Errorf("CreateClientTLSCert: Missing role")
	}
	if validity == 0 {
		validity = certs.DefaultClientValidityPeriod
	}
	certsDir := svc.env.CertsDir
	caCert, caKey, err := svc.env.GetCACert()
	cfg := svc.config
	privKey := utils.NewEd25519Key()
	// just a simple wrapper around the library
	x509Cert, err := utils.CreateClientCert(
		clientID, role,
		cfg.Country, cfg.Province, cfg.Locality, cfg.Org,
		validity, privKey.Public(), caCert, caKey)

	if err != nil {
		return nil, err
	}
	tlsCert = utils.X509CertToTLS(x509Cert, privKey)

	certPath := path.Join(certsDir, clientID+api.DefaultCertFileSuffix)
	keyPath := path.Join(certsDir, clientID+api.DefaultKeyFileSuffix)
	utils.SaveTLSCert(tlsCert, certPath, keyPath)

	return tlsCert, err
}

// Return a new server cert for use by hiveot gateway or device servers.
// If a provider is configured then ask the provider for a certificate, otherwise
// create a self-signed certificate.
//
//	serverName is used as the 'cn' in the certificate
//	address is the server address/hostname
//	validity of the certificate. 0 for default.
//	pubKey to include in the certificate, provided by the server
func (svc *CertsServiceImpl) CreateServerCert(
	serverName string, address string, validity time.Duration, pubKey crypto.PublicKey) (
	serverCert []*x509.Certificate, err error) {

	if serverName == "" {
		return nil, fmt.Errorf("CreateServerCert: Missing server name")
	}
	if pubKey == nil {
		return nil, fmt.Errorf("CreateServerCert: Missing public key")
	}
	if validity == 0 {
		validity = certs.DefaultServerValidityPeriod
	}
	names := []string{}

	caCert, caKey, err := svc.env.GetCACert()
	cfg := svc.config
	if address != "" {
		names = append(names, address)
	}

	if svc.provider != nil {
		// use the provider; must provide a public key?
		serverCert, err = svc.provider.CreateServerCert(
			serverName, names, validity, pubKey)

	} else if caCert != nil && caKey != nil {
		// create a self signed cert
		x509Cert, err2 := utils.CreateServerCert(
			serverName, certs.DefaultServerOU,
			cfg.Country, cfg.Province, cfg.Locality, cfg.Org,
			names, validity, pubKey, caCert, caKey)
		serverCert = []*x509.Certificate{x509Cert}
		err = err2
	}
	// if err == nil {
	// 	certPath := path.Join(svc.env.CertsDir, serverName+api.DefaultCertFileSuffix)
	// 	err = utils.SaveX509CertChain(serverCert, certPath)
	// }

	return serverCert, err
}

// GetCACert returns the self-signed x509 CA certificate from the environment
func (svc *CertsServiceImpl) GetCACert() *x509.Certificate {
	caCert, caKey, err := svc.env.GetCACert()
	_ = caKey
	_ = err
	return caCert
}

// Load the previously saved TLS certificate for the given clientID.
// Intended for local management of clients.
func (svc *CertsServiceImpl) GetClientTLSCert(clientID string) (*tls.Certificate, error) {

	tlsCert, err := svc.env.GetClientCert(clientID)
	return tlsCert, err
}

// GetServerCert returns a previously created server certificate.
//
// Return a saved cert or use a provider.
func (svc *CertsServiceImpl) GetServerCert(serverName string) (
	serverCert []*x509.Certificate, err error) {

	// certs can be provided out-of-band so always check for it.
	tlsCert, err := svc.env.GetServerCert(serverName)
	if err == nil && tlsCert != nil {
		serverCert, _ = utils.TLSCertToX509(tlsCert)
	}

	// if no cert is stored try the provider
	if serverCert == nil && svc.provider != nil {
		serverCert, err = svc.provider.GetServerCert(serverName)
	}
	return serverCert, err
}

// Refresh and save the admin client certificate.
// This saves the cert to file and updates the app environment.
//
//	validity is the period for new certificate.. 0 for default.
//	minValidity is the minimum time the old certificate has to be valid. 0 for 20% of validity.
func (svc *CertsServiceImpl) RefreshAdminCert(
	validity time.Duration, minValidity time.Duration) error {

	if validity == 0 {
		validity = certs.DefaultAdminValidityPeriod
	}
	if minValidity == 0 {
		minValidity = validity / 5
	}

	certsDir := svc.env.CertsDir

	// create a new admin client cert if validation period is set and no exist cert exists
	adminID := api.DefaultAdminUserID
	adminCert, err := svc.env.GetClientCert(adminID)

	if err != nil || adminCert == nil {
		// admin client cert doesn't exist, create one
		certPath := filepath.Join(certsDir, adminID+api.DefaultCertFileSuffix)
		keyPath := filepath.Join(certsDir, adminID+api.DefaultKeyFileSuffix)

		privKey := utils.NewEd25519Key()
		adminX509, err2 := svc.CreateClientCert(
			adminID, api.ClientRoleAdmin, validity, privKey.Public())
		err = err2
		if err == nil {
			err = utils.SaveX509Cert(adminX509, certPath)
		}
		if err == nil {
			err = utils.SavePrivateKey(privKey, keyPath)
		}
	}
	return err
}

// Refresh the self signed CA before it expires.
//
// This saves the cert to file and updates the app environment.
//
// If no CA exists then create a new one using the configured private key.
// If no private key is configured then create and save one.
//
// An error is returned if refresh was attempted and failed.
//
//	validity is the validity period of new certificates or 0 for default
//	minValidity is the minimum remaining validity before renewing. 0 for 20%.
func (svc *CertsServiceImpl) RefreshCA(
	validity time.Duration, minValidity time.Duration) (err error) {

	if validity == 0 {
		validity = certs.DefaultCAValidityPeriod
	}
	if minValidity == 0 {
		minValidity = validity / 5
	}

	env := svc.env

	// todo: should lifespan be a config? for now use 20% as the threshold
	minValidUntil := time.Now().Add(minValidity)
	// caKeyPath := path.Join(certsDir, api.DefaultCaKeyFile)
	// caCertPath := path.Join(certsDir, api.DefaultCaCertFile)

	// caCert, err := utils.LoadCACert(caCertPath)
	// caKey, err := utils.LoadPrivateKey(caKeyPath)
	caCert, caKey, err := env.GetCACert()

	// if a private key doesn't exist, create and save a new one
	if caKey == nil {
		caKey = utils.NewEd25519Key()
		// if a CA cert was known it is no longer valid
		caCert = nil

		// slog.Warn("RefreshCA: New CA key", "caKeyPath", caKeyPath)
		// err = utils.SavePrivateKey(caKey, caKeyPath)
	}

	// if the CA doesn't exist, create a new one, otherwise use it as a template.
	if caCert == nil {
		caCert, caKey, err = CreateSelfSignedCACert(caKey, validity)

		// slog.Warn("RefreshCA: New CA at", "caCertPath", caCertPath)
		// err = utils.SaveX509Cert(caCert, caCertPath)
		env.SetCACert(caCert, caKey)

	} else if caCert.NotAfter.Before(minValidUntil) {
		template := *caCert
		newCA, err := utils.CreateCertFromTemplate(&template, validity, caCert, caKey)

		if err != nil {
			return err
		}
		caCert = newCA
		// if err == nil {
		// err = utils.SaveX509Cert(newCA, caCertPath)
		// }
		env.SetCACert(newCA, caKey)
	}
	return nil
}

// Refresh and save the server certificate if needed.
//
// This uses the server cert from the app environment and updates it
// if its validity period is below minRemaining.
// When a new certificate is created it is saved to the environment certs directory.
//
//	serverName is the certificate. Use "" for the default.
//	validity is the validity period of new certificates. 0 for default.
//	minRemaining is the minimum time the certificate should still be valid before renewing. 0 for 20%.
func (svc *CertsServiceImpl) RefreshServerCert(
	serverName string, validity time.Duration, minRemaining time.Duration) (err error) {

	var serverKey crypto.Signer
	if validity == 0 {
		validity = certs.DefaultServerValidityPeriod
	}
	if minRemaining == 0 {
		minRemaining = validity / 5
	}
	if serverName == "" {
		serverName = api.DefaultServerName
	}

	minValidUntil := time.Now().Add(minRemaining)

	// this can be the cached self-signed or provider cert with key

	// validate the cert if available
	tlsCert, err := svc.env.GetServerCert(serverName)
	if err == nil {
		serverKey, _ = tlsCert.PrivateKey.(crypto.Signer)
		oldCert, err := x509.ParseCertificate(tlsCert.Certificate[0])
		if err == nil {
			// if the cert is valid beyond minValidUntil then we're good
			if !oldCert.NotAfter.Before(minValidUntil) {
				return nil
			}
		}
	}

	// renew and save the certificate
	var serverChain []*x509.Certificate

	if serverKey == nil {
		serverKey = utils.NewEd25519Key()
	}

	if svc.provider != nil {
		serverChain, err = svc.provider.Refresh(serverName)
	} else {
		// self signed
		ip := utils.GetOutboundIP("")
		serverChain, err = svc.CreateServerCert(
			serverName, ip.String(), validity, serverKey.Public())

		// newCert, err = utils.CreateCertFromTemplate(
		// 	oldCert, validityPeriod, env.CaCert, svc.config.CaKey)
	}
	if err != nil {
		return err
	}
	serverCert := utils.X509ChainToTLS(serverChain, serverKey)
	svc.env.SetServerCert(serverCert)
	return err
}

// Start publishes the td when app is ready
func (svc *CertsServiceImpl) Start() {
	tdJson := string(certs.CertsServiceTD)
	tdoc, err := wot.UnmarshalTD(tdJson)
	if err == nil {
		tdoc.ID = svc.GetID()
		tdoc.SetType(certs.CertsServiceCellType)
		svc.WriteTD(tdoc)
	}
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
		Roots:     svc.env.GetRootCAs().Clone(),
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	_, err = clientCert.Verify(opts)

	return err
}

// Create a ready-to-use self-signed certificate provider.
//
// This:
//   - creates a self-signed CA certificate if one doesn't exist
//   - creates a server certificate if one doesn't exist
//   - creates a client certificate if one doesn't exist
//
// Call Start to publish its TD.
func NewCertsServiceImpl(env *api.AppEnvironment, config *certs.CertsConfig) (*CertsServiceImpl, error) {
	var err error

	thingID := certs.CertsServiceDefaultThingID
	if config.ThingID != "" {
		thingID = config.ThingID
	}

	svc := &CertsServiceImpl{
		ExposedThing: thing.NewExposedThing(thingID, nil),
		config:       config,
		env:          env,
	}
	slog.Info("NewCertsServiceImpl: Starting certs service")

	// initialize all ceritificates with defaults
	err = svc.RefreshCA(config.CACertValidity, 0)
	if err == nil {
		err = svc.RefreshServerCert("", config.ServerCertValidity, 0)
	}
	if err == nil {
		err = svc.RefreshAdminCert(config.AdminCertValidity, 0)
	}

	// // include the CA in the certificate pool for verifying certificates
	// svc.caCertPool, _ = x509.SystemCertPool()
	// svc.caCertPool.AddCert(svc.config.CaCert)

	// // create an admin client cert if validation period is set and no exist cert exists
	// if config.AdminCertValidityDays > 0 {
	// 	err = svc.CreateAdminCert()
	// }
	var _ certs.ICertsService = svc // api validation
	return svc, err
}

package service

import (
	"crypto"
	"crypto/tls"
	"crypto/x509"
	_ "embed"
	"os"
	"path"

	"github.com/hiveot/hivekit/go/modules"
	certsapi "github.com/hiveot/hivekit/go/modules/certs/api"
	"github.com/hiveot/hivekit/go/modules/certs/certutils"
)

// CertsService is a module for managing certificates.
// This implements IHiveModule and ICertsServer interfaces.
//
// The module can be accessed:
//  1. Natively from golang. The module supports the ICertsService interface.
//  2. Using hivekit RRN messaging (request-response-notification). See CertsMsgHandler.go
//
// # See certs-tm.json for the WoT TM definition of the module.
type CertsService struct {
	// base forwards unhandled requests and notifications
	modules.HiveModuleBase

	// ca certificate or nil if none found
	caCert *x509.Certificate
	// ca key-pair
	caPrivKey crypto.PrivateKey

	// the default server certificate as shared between modules
	defaultServerTlsCert *tls.Certificate

	// the RRN messaging API
	msgHandler *CertsMsgHandler

	// directory where certificates are stored
	certsDir string
}

// GetTM returns the module TM document
// It includes forms for messaging access through the WoT.
func (m *CertsService) GetTM() string {
	tmJson := CertsTMJson
	return string(tmJson)
}

// Start readies the certificate management module for use.
//
// This loads the stored CA or creates a self-signed if none is found
// This loads the default TLS certificate for use by servers or create a new if one isnt found
func (m *CertsService) Start() (err error) {

	caCertPath := path.Join(m.certsDir, certsapi.DefaultCaCertFile)
	caKeyPath := path.Join(m.certsDir, certsapi.DefaultCaKeyFile)
	if m.certsDir != "" {
		m.caCert, m.caPrivKey, err = certutils.LoadCA(caCertPath, caKeyPath)

		// Load a saved default certificate
		if m.defaultServerTlsCert == nil {
			m.defaultServerTlsCert, err = m.LoadServerCert(certsapi.DefaultServerName)
		}
	}
	// create missing CA key and cert
	if m.caCert == nil || m.caPrivKey == nil {
		// Make a clean start with cert and key.
		_ = os.Remove(caCertPath)
		_ = os.Remove(caKeyPath)
		m.caCert, m.caPrivKey, err = m.CreateCACert()
	}
	// create missing default server certificate
	// FIXME: validate the certificate is expired
	if m.defaultServerTlsCert == nil {
		m.defaultServerTlsCert, err = m.CreateServerCert(
			certsapi.DefaultServerName, "", nil, nil)
	}

	m.msgHandler = NewCertsMsgHandler(m.GetModuleID(), m)
	m.SetRequestHook(m.msgHandler.HandleRequest)
	return err
}

// Stop any running actions
func (m *CertsService) Stop() {
	// m.service.Stop()
}

// Create a new certificate service module
// certsDir is the storage directory to read or create keys and certificates.
func NewCertsService(certsDir string) *CertsService {
	m := &CertsService{
		certsDir: certsDir,
	}
	m.SetModuleID(certsapi.DefaultCertsServiceThingID)
	var _ modules.IHiveModule = m    // interface check
	var _ certsapi.ICertsService = m // interface check
	return m
}

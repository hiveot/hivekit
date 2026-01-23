package module

import (
	"crypto"
	"crypto/tls"
	"crypto/x509"
	_ "embed"
	"os"
	"path"

	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/certs"
	"github.com/hiveot/hivekit/go/modules/certs/certutils"
	"github.com/hiveot/hivekit/go/modules/certs/server"
)

// CertsModule is a module for managing certificates.
// This implements IHiveModule and ICertsService interfaces.
//
// The module can be accessed:
//  1. Natively from golang. The module supports the ICertsService interface.
//  2. Using hivekit RRN messaging (request-response-notification). See CertsMsgHandler.go
//
// # See certs-tm.json for the WoT TM definition of the module.
type CertsModule struct {
	// base forwards unhandled requests and notifications
	modules.HiveModuleBase

	// ca certificate or nil if none found
	caCert *x509.Certificate
	// ca key-pair
	caPrivKey crypto.PrivateKey

	// the default server certificate as shared between modules
	defaultServerCert *tls.Certificate

	// the RRN messaging API
	msgHandler *server.CertsMsgHandler

	// directory where certificates are stored
	certsDir string
}

// GetTM returns the module TM document
// It includes forms for messaging access through the WoT.
func (m *CertsModule) GetTM() string {
	tmJson := server.CertsTMJson
	return string(tmJson)
}

// Start readies the certificate management module for use.
func (m *CertsModule) Start() (err error) {

	caCertPath := path.Join(m.certsDir, certs.DefaultCaCertName)
	caKeyPath := path.Join(m.certsDir, certs.DefaultCaKeyName)
	if m.certsDir != "" {
		m.caCert, m.caPrivKey, err = certutils.LoadCA(caCertPath, caKeyPath)

		// Load a saved default certificate
		if m.defaultServerCert == nil {
			m.defaultServerCert, err = m.LoadServerCert(certs.DefaultServerName)
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
	if m.defaultServerCert == nil {
		m.defaultServerCert, err = m.CreateServerCert(
			certs.DefaultServerName, "", nil, nil)
	}

	m.msgHandler = server.NewCertsMsgHandler(m.GetModuleID(), m)
	return err
}

// Stop any running actions
func (m *CertsModule) Stop() {
	// m.service.Stop()
}

// Create a new certificate management module
// certsDir is the storage directory to read or create keys and certificates.
func NewCertsModule(certsDir string) *CertsModule {
	m := &CertsModule{
		certsDir: certsDir,
	}
	m.Init(certs.DefaultCertsThingID, nil)
	var _ modules.IHiveModule = m // interface check
	return m
}

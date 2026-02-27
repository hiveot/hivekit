package certsserver

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
)

// CertsServer is a module for managing certificates.
// This implements IHiveModule and ICertsServer interfaces.
//
// The module can be accessed:
//  1. Natively from golang. The module supports the ICertsService interface.
//  2. Using hivekit RRN messaging (request-response-notification). See CertsMsgHandler.go
//
// # See certs-tm.json for the WoT TM definition of the module.
type CertsServer struct {
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
func (m *CertsServer) GetTM() string {
	tmJson := CertsTMJson
	return string(tmJson)
}

// Start readies the certificate management module for use.
//
// yamlConfig : todo
func (m *CertsServer) Start(yamlConfig string) (err error) {

	caCertPath := path.Join(m.certsDir, certs.DefaultCaCertName)
	caKeyPath := path.Join(m.certsDir, certs.DefaultCaKeyName)
	if m.certsDir != "" {
		m.caCert, m.caPrivKey, err = certutils.LoadCA(caCertPath, caKeyPath)

		// Load a saved default certificate
		if m.defaultServerTlsCert == nil {
			m.defaultServerTlsCert, err = m.LoadServerCert(certs.DefaultServerName)
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
	if m.defaultServerTlsCert == nil {
		m.defaultServerTlsCert, err = m.CreateServerCert(
			certs.DefaultServerName, "", nil, nil)
	}

	m.msgHandler = NewCertsMsgHandler(m.GetModuleID(), m)
	m.SetRequestHook(m.msgHandler.HandleRequest)
	return err
}

// Stop any running actions
func (m *CertsServer) Stop() {
	// m.service.Stop()
}

// Create a new certificate server module
// certsDir is the storage directory to read or create keys and certificates.
func NewCertsServer(certsDir string) *CertsServer {
	m := &CertsServer{
		certsDir: certsDir,
	}
	m.SetModuleID(certs.DefaultCertsThingID)
	var _ modules.IHiveModule = m // interface check
	var _ certs.ICertsServer = m  // interface check
	return m
}

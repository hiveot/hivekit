package service

import (
	"crypto"
	"crypto/tls"
	"crypto/x509"
	_ "embed"
	"fmt"
	"log/slog"
	"os"
	"path"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/certs"
	"github.com/hiveot/hivekit/go/modules/certs/internal/providers/selfsigned"
	"github.com/hiveot/hivekit/go/utils"
)

// Embed the certs TM
//
//go:embed "certs-tm.json"
var CertsTMJson []byte

// Defaults for a self-signed CA
const DefaultCA_CN = "HiveOT"
const DefaultCA_Country = "Earth"
const DefaultCA_Locality = "HiveOT"
const DefaultCA_Org = "Internet of things"
const DefaultCA_Province = "One World"
const DefaultCA_Validity = 365*20 + 5

// CertsServiceImpl is a module for managing certificates.
// This implements IHiveModule and ICertsServer interfaces.
//
// The module can be accessed:
//  1. Natively from golang. The module supports the ICertsService interface.
//  2. Using hivekit RRN messaging (request-response-notification). See CertsMsgHandler.go
//
// # See certs-tm.json for the WoT TM definition of the module.
type CertsServiceImpl struct {
	// base forwards unhandled requests and notifications
	*modules.HiveModuleBase

	// ca certificate or nil if none found
	caCert *x509.Certificate
	// ca key-pair
	caPrivKey crypto.PrivateKey

	// service configuration
	config certs.CertsConfig
}

// Invoke the GetCACert method
func (svc *CertsServiceImpl) _handleGetCACert(req *msg.RequestMessage) (resp *msg.ResponseMessage, err error) {
	// no args
	cert, err := svc.GetCACert()
	if err != nil {
		return nil, err
	}
	// convert cert to PEM
	caPEM := utils.X509CertToPEM(cert)
	resp = req.CreateResponse(caPEM, err)
	return resp, nil
}

// Decode the Get Server cert method
func (svc *CertsServiceImpl) _handleGetServerCert(req *msg.RequestMessage) (resp *msg.ResponseMessage, err error) {
	// no args
	var serverName string
	req.Decode(&serverName)
	cert, err := svc.LoadServerCert(serverName)
	if err != nil {
		return nil, err
	}
	// convert cert to PEM
	certPEM := utils.X509CertToPEM(cert)
	resp = req.CreateResponse(certPEM, err)
	return resp, nil
}

// Create and save a HiveOT self-signed CA certificate and keys.
//
// If a directory is configured, save the CA in the directory.
// If the directory already contains a CA then do nothing and return an error.
// If the directory contains a key-pair then use it instead of creating a new one.
// If no key-pair is provided this uses ED25519 keys, as browsers nowadays support ED25519 (2026)
//
// validityDays is the CA's validity in days
// This returns the CA, key or an error
func (svc *CertsServiceImpl) CreateCACert() (
	caCert *x509.Certificate, privKey crypto.PrivateKey, err error) {

	caCert, privKey, _, err = selfsigned.CreateSelfSignedCA(
		DefaultCA_Country,
		DefaultCA_Province,
		DefaultCA_Locality,
		DefaultCA_Org,
		DefaultCA_CN,
		DefaultCA_Validity,
		utils.KeyTypeED25519)

	if svc.config.CertsDir != "" {
		// save the CA, but only if it won't overwrite an existing certificate
		caCertPath := path.Join(svc.config.CertsDir, api.DefaultCaCertFile)
		caKeyPath := path.Join(svc.config.CertsDir, api.DefaultCaKeyFile)

		if _, err := os.Stat(caCertPath); err == nil {
			err = fmt.Errorf("the CA certificate exists at %s", caCertPath)
			return nil, nil, err
		}
		if err == nil {
			err = utils.SavePrivateKey(privKey, caKeyPath)
		}
		if err == nil {
			err = utils.SaveX509CertToPEM(caCert, caCertPath)
		}
	}
	return caCert, privKey, err
}

// Create and save a TLS server certificate for the module with the given ID.
// localhost, 127.0.0.1 and the given hostname are added to the SAN names.
// If the certificate exists it is replaced.
//
// While the server public key can be obtained from the private key, the crypto API
// does not offer this method so it has to be provided separately.
//
// cn is the domain name or owner ID of the certificate and the name under which the certificate is saved.
// hostname will be added to the certificate SAN. If omitted, the outbound IP will be used.
// serverPrivKey is the server's private key. nil to generate a ecdsa key-pair.
// serverPubKey is embedded in the server certificate
//
// The certificate will be signed by the CA on file, if present.
// If LetsEncrypt is configured then an internet connection is required. (a future feature)
func (svc *CertsServiceImpl) CreateServerCert(
	serverName string, hostname string,
	serverPrivKey crypto.PrivateKey, serverPubKey crypto.PublicKey) (
	tlsCert *tls.Certificate, err error) {

	// names are the SAN names to include with the certificate, localhost and 127.0.0.1 are always added
	names := []string{}
	if hostname != "" {
		names = append(names, hostname)
	} else {
		ip := utils.GetOutboundIP("")
		names = append(names, ip.String())
	}
	if serverPrivKey == nil {
		serverPrivKey, serverPubKey = utils.NewEcdsaKey()
	}
	// use self-signed CA until letsencrypt is supported
	serverCert, err := selfsigned.CreateSelfSignedServerCert(
		serverName, DefaultCA_Org, 365,
		serverPubKey, names, svc.caCert, svc.caPrivKey)
	if err != nil {
		return tlsCert, err
	}
	tlsCert = utils.X509CertToTLS(serverCert, serverPrivKey)

	// persist the certificate
	certPath := path.Join(svc.config.CertsDir, serverName+"Cert.pem")
	keyPath := path.Join(svc.config.CertsDir, serverName+"Key.pem")
	err = utils.SaveTLSCertToPEM(tlsCert, certPath, keyPath)

	return tlsCert, err
}

// Return the configured CA certificate
func (svc *CertsServiceImpl) GetCACert() (*x509.Certificate, error) {
	if svc.caCert == nil {
		return nil, fmt.Errorf("service not initialized")
	}
	return svc.caCert, nil
}

// GetTM returns the module TM document
// It includes forms for messaging access through the WoT.
func (svc *CertsServiceImpl) GetTM() string {
	tmJson := CertsTMJson
	return string(tmJson)
}

// HandleRequest for properties or actions
// If the request is not recognized nil is returned.
// If the request is missing the sender, an error is returned
func (svc *CertsServiceImpl) HandleRequest(
	req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {

	if req.ThingID != svc.GetThingID() {
		return svc.ForwardRequest(req, replyTo)
	}

	var resp *msg.ResponseMessage
	if req.SenderID == "" {
		// todo: is this really needed?
		err = fmt.Errorf("missing senderID in request")
	} else if req.Operation == td.OpInvokeAction {
		// certificate specific operations
		switch req.Name {
		case certs.GetCACertAction:
			resp, err = svc._handleGetCACert(req)
		case certs.GetServerCertAction:
			resp, err = svc._handleGetServerCert(req)
		default:
			err = fmt.Errorf("Unknown request name '%s' for thingID '%s'", req.Name, req.ThingID)
		}
	} else {
		err = fmt.Errorf("Unsupported operation '%s' for thingID '%s'", req.Operation, req.ThingID)
	}
	if resp != nil {
		err = replyTo(resp)
	}
	return err
}

// LoadServerCert loads the previously saved public server certificate from the
// certificate directory.
// The file names used are {serverName}Cert.pem
func (svc *CertsServiceImpl) LoadServerCert(
	serverName string) (cert *x509.Certificate, err error) {

	if svc.config.CertsDir == "" {
		return nil, fmt.Errorf("certificate directory is not configured")
	}
	serverCertPath := path.Join(svc.config.CertsDir, serverName+"Cert.pem")
	cert, err = utils.LoadX509CertFromPEM(serverCertPath)

	return cert, err
}

// LoadServerTLSCert loads a previously saved server certificate with key from the
// certificate directory.
// The file names used are {serverName}Cert.pem and {serverName}Key.pem
func (svc *CertsServiceImpl) LoadServerTLSCert(serverName string) (
	serverCert *tls.Certificate, err error) {

	if svc.config.CertsDir == "" {
		return serverCert, fmt.Errorf("certificate directory is not configured")
	}
	serverCertPath := path.Join(svc.config.CertsDir, serverName+"Cert.pem")
	serverKeyPath := path.Join(svc.config.CertsDir, serverName+"Key.pem")
	serverCert, err = utils.LoadTLSCertFromPEM(serverCertPath, serverKeyPath)

	return serverCert, err
}

// Start readies the certificate management module for use.
//
// This loads the stored CA or creates a self-signed if none is found
// This loads the default TLS certificate for use by servers or create a new if one isnt found
func (svc *CertsServiceImpl) Start() (err error) {
	slog.Info("Start: Starting certs service")

	if svc.config.CertsDir == "" {
		return fmt.Errorf("Start: Missing certificate directory in config")
	}

	caCertPath := path.Join(svc.config.CertsDir, api.DefaultCaCertFile)
	caKeyPath := path.Join(svc.config.CertsDir, api.DefaultCaKeyFile)

	svc.caCert, svc.caPrivKey, err = utils.LoadCA(caCertPath, caKeyPath)

	// Load a server cert if specified, load it
	// if svc.config.ServerCertName != "" {
	// 	svc.serverTlsCert, err = svc.LoadServerCert(svc.config.ServerCertName)
	// }
	// create missing CA key and cert
	if svc.caCert == nil || svc.caPrivKey == nil {
		// Make a clean start with cert and key.
		_ = os.Remove(caCertPath)
		_ = os.Remove(caKeyPath)
		svc.caCert, svc.caPrivKey, err = svc.CreateCACert()
	}
	// create a new default server certificate
	// FIXME: validate the certificate is expired
	// if svc.config.CreateServerCertName != "" {
	// 	svc.serverTlsCert, err = svc.CreateServerCert(
	// 		svc.config.ServerCertName, "", nil, nil)
	// }
	return err
}

// Stop any running actions
func (svc *CertsServiceImpl) Stop() {
	slog.Info("Stop: Stopping certs service")
	// m.service.Stop()
}

func (svc *CertsServiceImpl) VerifyCert(serverName string, cert *x509.Certificate) (err error) {
	cn, err := selfsigned.VerifyCert(cert, svc.caCert)
	if err == nil {
		if cn != serverName {
			err = fmt.Errorf("expected cn to be '%s' but it is '%s' instead", serverName, cn)
		}
	}
	return err
}

// Create a new certificate service module
// certsDir is the storage directory to read or create keys and certificates.
func NewCertsServiceImpl(config certs.CertsConfig) *CertsServiceImpl {
	// certificate service is a singleton
	thingID := certs.DefaultCertsServiceThingID
	m := &CertsServiceImpl{
		config:         config,
		HiveModuleBase: modules.NewHiveModuleBase(thingID, 0),
	}
	var _ api.IHiveModule = m     // interface check
	var _ certs.ICertsService = m // interface check
	return m
}

package certs

import (
	"crypto/tls"
	"crypto/x509"

	"github.com/hiveot/hivekit/go/modules/services/certs/keys"
)

// DefaultCertsThingID is the default thingID of the certs module.
const DefaultCertsThingID = "certs"

const DefaultCaCertName = "caCert.pem"
const DefaultCaKeyName = "caKey.pem"

// DefaultServerName is the name of the shared default server cert
const DefaultServerName = "server"

// ICertsService interface of the certificate service
type ICertsService interface {

	// Create and store the server TLS certificate for a server module.
	//
	// This includes localhost and 127.0.0.1 in the certificate SAN names.
	// A server private key can be provided or will be created when omitted.
	// This returns a TLS certificate, signed by the service CA.
	// If the service is configured to use LetsEncrypt, then a working internet is
	// required to have LetsEncrypt create the certificate.
	//
	// While the default serverKey is ecdsa it is also possible to use RSA or ed25519.
	//
	// moduleName is the name under which to store the key and certificate.
	// hostname is the name or IP to include in the certificate SAN. "" to ignore.
	// serverKey is the server key used to create the certificate. nil to generate.
	CreateServerCert(moduleID string, hostname string, serverKey keys.IHiveKey) (*tls.Certificate, error)

	// GetCACert returns the x509 CA certificate.
	// Returns and error if a CA is not initialized or can not be returned.
	GetCACert() (*x509.Certificate, error)

	// Return the default shared (between modules) server certificate.
	//
	GetDefaultServerCert() (*tls.Certificate, error)

	// LoadServerCert loads a previously save server certificate from the
	// certificate directory.
	// This returns an error if certificate/key files are not found.
	//
	// moduleID whose certificate to retrieve
	LoadServerCert(moduleID string) (*tls.Certificate, error)

	// Verify if the given certificate belongs to the module and is signed by the CA
	// This returns an error if the certificate cannot be verified or doesn't
	// have the moduleID as cn.
	VerifyCert(moduleID string, cert *x509.Certificate) error
}

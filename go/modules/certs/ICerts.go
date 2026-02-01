package certs

import (
	"crypto"
	"crypto/tls"
	"crypto/x509"
)

// DefaultCertsThingID is the default thingID of the certs module.
const DefaultCertsThingID = "certs"

const DefaultCaCertName = "caCert.pem"
const DefaultCaKeyName = "caKey.pem"

// DefaultServerName is the name of the shared default server cert
const DefaultServerName = "server"
const DefaultServerCertName = DefaultServerName + "Cert.pem"
const DefaultServerKeyName = DefaultServerName + "Key.pem"

// RRN Actions
const (
	ActionGetCACert     = "getCACert"
	ActionGetServerCert = "getServerCert"
)

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
	// While the default serverKey is ecdsa it is also possible to use ed25519.
	// Since the crypto api does not offer a method to obtain the public key from
	// the private key, it has to be provided separately.
	//
	// moduleName is the name under which to store the key and certificate.
	// hostname is the name or IP to include in the certificate SAN. "" to ignore.
	// serverPrivKey is the server private key used to create the TLS certificate. nil to generate.
	// serverPubKey is the corresponding public key.
	CreateServerCert(moduleID string, hostname string,
		serverPrivKey crypto.PrivateKey, serverPubKey crypto.PublicKey) (*tls.Certificate, error)

	// GetCACert returns the x509 CA certificate.
	// Returns and error if a CA is not initialized or can not be returned.
	GetCACert() (*x509.Certificate, error)

	// Return the default public server certificate.
	GetDefaultServerCert() (*x509.Certificate, error)

	// Return the default shared (between modules) server certificate.
	//
	GetDefaultServerTlsCert() (*tls.Certificate, error)

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

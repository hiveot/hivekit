package certs

import (
	"crypto"
	"crypto/tls"
	"crypto/x509"

	"github.com/hiveot/hivekit/go/api"
)

// Virtual module to initialize the certificates needed to run the factory.
// This creates self-signed certificates if not loaded.
const InitFactoryCertsModuleType = "initFactoryCerts"

// certs service module type for factory. Must implement ICertsService
const CertsServerModuleType = "certs"

// DefaultCertsServiceThingID is the default thingID of the certs service module.
const DefaultCertsServiceThingID = "certs"

// [deprecated] DefaultServerName is the name of the shared default server cert
// const DefaultServerName = "server"
// const DefaultServerCertFile = DefaultServerName + "Cert.pem"
// const DefaultServerKeyFile = DefaultServerName + "Key.pem"

// RRN Actions
const (
	GetCACertAction = "getCACert"

	// the get ServerCert action
	// input: server name
	GetServerCertAction = "getServerCert"
)

// CertsConfig defines certificate service configuration.
// This can also be provided through the factory function
type CertsConfig struct {
	// The certificate storage directory. Required.
	CertsDir string `yaml:"certsDir"`

	// Create a server certificate on startup.
	// This requires ServerCertName to be set.
	// CreateServerCert bool `yaml:"createServerCert,omitempty"`

	// Provider of the Certificate service. "selfsigned" (default) or "letsencrypt".
	// Lets-encrypt support is not complete yet so nothing to see here.
	Provider string `yaml:"provider,omitempty"`

	// ServerCertName holds the default name of the server certificate to load on request.
	// If empty no server certificate will be available.
	// ServerCertName string `yaml:"createServerCertName,omitempty"`
}

// ICertsService interface of the certificate module server
type ICertsService interface {
	api.IHiveModule

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
	//  serverName is the name under which to store the key and certificate.
	//  hostName is the name or IP to include in the certificate SAN. "" to ignore.
	//  serverPrivKey is the server private key used to create the TLS certificate. nil to generate.
	//  serverPubKey is the corresponding public key.
	//
	// Only administrators are allowed to create certificates.
	CreateServerCert(serverName string, hostName string,
		serverPrivKey crypto.PrivateKey, serverPubKey crypto.PublicKey) (*tls.Certificate, error)

	// GetCACert returns the x509 CA certificate.
	// Returns and error if a CA is not initialized or can not be returned.
	GetCACert() (*x509.Certificate, error)

	// LoadServerCert loads the public x509 certification of a previous saved certificate.
	//
	// serverName name under which certificate was created/saved
	// Only administrators or modules themselves are allowed to load the TLS certificate.
	LoadServerCert(serverName string) (*x509.Certificate, error)

	// LoadServerTLSCert loads a previously saved server certificate from the
	// certificate directory.
	// This returns an error if certificate/key files are not found.
	//
	// serverName name under which certificate was created/saved
	// Only administrators or modules themselves are allowed to load the TLS certificate.
	LoadServerTLSCert(serverName string) (*tls.Certificate, error)

	// Verify if the given certificate is signed by the CA.
	//
	// This checks if the certificate uses serverName as the Common Name.
	//
	// This returns an error if the certificate cannot be verified or doesn't
	// have the serverName as cn.
	VerifyCert(serverName string, cert *x509.Certificate) error
}

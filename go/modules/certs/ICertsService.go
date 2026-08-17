package certs

import (
	"crypto"
	"crypto/x509"
	"time"

	"github.com/hiveot/hivekit/go/api"
)

// Virtual module to initialize the certificates needed to run the factory.
// This creates self-signed certificates if not loaded.
const InitFactoryCertsModuleType = "initFactoryCerts"

// certs service module type for factory. Must implement ICertsService
const CertsServiceModuleType = "certs"

// DefaultCertsServiceThingID is the default thingID of the certs service module.
const DefaultCertsServiceThingID = "certs"

// Default certificate validity periods
const (
	DefaultAdminID  = "admin"
	DefaultServerOU = ""

	DefaultAdminValidityPeriod  = time.Hour * 24 * 90
	DefaultClientValidityPeriod = time.Hour * 24 * 90
	DefaultCAValidityPeriod     = time.Hour * 24 * 90
	DefaultServerValidityPeriod = time.Hour * 24 * 90
)

// [deprecated] DefaultServerName is the name of the shared default server cert
// const DefaultServerName = "server"
// const DefaultServerCertFile = DefaultServerName + "Cert.pem"
// const DefaultServerKeyFile = DefaultServerName + "Key.pem"

const SelfSignedProvider = "selfsigned"

const LetsEncryptProvider = "letsencrypt"

// RRN Actions
const (
	GetCACertAction = "getCACert"

	// the get ServerCert action
	// input: server name
	GetServerCertAction = "getServerCert"

	// verify the client certificate
	// input: PEM encoded client certificate
	VerifyClientCertAction = "verifyClientCert"
)

// CertsConfig defines certificate service configuration.
// This can also be provided through the factory function
type CertsConfig struct {
	// The certificate storage directory. Required.
	CertsDir string `yaml:"certsDir"`

	// The self-signed CA certificate to use.
	CaCert *x509.Certificate
	// The private/public key-pair of the self-signed CA
	CaKey crypto.Signer

	// Override the default settings for Country, Locality, Org and Province
	Country  string
	Locality string
	Org      string
	Province string
}

// ICertsProvider interface of the certificate provider service like letsencrypt
type ICertProvider interface {
	// Create a new server certficate.
	//
	// If a provider is used then request it from the provider.
	// If no provider is used then create a self-signed certificate.
	//
	//  serverName is the name of the server under which the cert, key and certificate is stored.
	//  names are the IP or domain names to include in the certificate SAN. nil to ignore.
	//	validity is the validity period of the certificate. Required.
	//  serverPubKey is the server public key to include in the certificate. Required.
	//
	// This returns the server x509 certificate chain.
	CreateServerCert(
		serverName string, names []string,
		validity time.Duration, pubKey crypto.PublicKey) (
		[]*x509.Certificate, error)

	// GetServerCert returns a previously created server certificate.
	//
	// If a cert exists in the local certs storage, load and return it. This can be
	// created and stored out of band. The name is {certs}/{serverName}Cert.pem.
	//
	// If no cert is found and a provider is used then request the cert from the provider.
	GetServerCert(serverName string) ([]*x509.Certificate, error)

	// Refresh the server certificate.
	//
	// This returns the new certificate.
	Refresh(serverName string) ([]*x509.Certificate, error)
}

// ICertsService interface of the certificate management service
type ICertsService interface {
	api.IHiveModule

	// Create a new server certificate chain
	//
	// This includes localhost and 127.0.0.1 in the certificate SAN names.
	// A server private key can be provided or will be created when omitted.
	// This returns a TLS certificate, signed by the provider CA.
	// Providers like Lets-Encrypt require a working internet to create the certificate.
	//
	// The country, province, locality fields are set to hiveot's default from
	// the service configuration.
	//
	//  serverName is the name of the server and under which to store the key and certificate.
	//  hostName is the name or IP to include in the certificate SAN. "" to ignore.
	//	validity is the validity period of the certificate or 0 for the recommended default.
	//  serverPubKey is the server public key to include in the certificate.
	//
	// This returns the server x509 certificate chain.
	// Use utils.X509CertToTLS to convert it to a TLS certificate.
	CreateServerCert(
		serverName string, hostName string,
		validity time.Duration, pubKey crypto.PublicKey) (
		[]*x509.Certificate, error)

	// Create a self-signed client certificate using the given client public key.
	//
	// Intended for devices and consumers to support authentication with a server
	// using a client certificate. The country, province, locality fields are set
	// to hiveot's default from the service configuration.
	//
	// Use utils.X509CertToTLS to convert it to a TLS certificate.
	//
	//	clientID identifies the authentication accountID of the client
	//  ou is intended to identify the client as a device, consumer or service
	//	validity is the time the certificate is valid for
	//	pubKey is the client's public key needed to authenticate.
	CreateClientCert(clientID string, ou string, validity time.Duration,
		pubKey crypto.PublicKey) (*x509.Certificate, error)

	// GetCACert returns the service self-signed CA certificate.
	GetCACert() *x509.Certificate

	// GetServerCert returns a previously created server certificate chain.
	// This first locates a previously saved certificate with the name in the
	// certs directory.
	// If no certificate was found then check with the configured providers.
	GetServerCert(serverName string) ([]*x509.Certificate, error)

	// Refresh the server certificate if needed.
	// The certificate is updated when its remaining validity is below minRemaining
	// If minRemaining is negative then force a refresh.
	// This returns the new certificate if refreshed or the old certificate if plenty of life is left.
	Refresh(serverName string, minRemaining time.Duration) ([]*x509.Certificate, error)

	// Verify if the given client certificate is valid and signed by the CA.
	//
	// Intended for validating self-signed client certificates.
	//
	// This returns an error if the certificate cannot be verified or doesn't
	// have the clientID as cn.
	VerifyClientCert(clientID string, clientCert *x509.Certificate) error
}

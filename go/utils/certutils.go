// Package utils with certificate management helpers
package utils

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"log/slog"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

// Certificate Organization Unit for client certificate based authorization.
// Intended to identify the purpose of the certificate.
const (
	//ClientOUAdmin lets a client approve things provisioning (postOOB), add and remove users
	ClientOUAdmin = "admin"

	// OUNone is the default OU with no API access permissions
	ClientOUNone = "n/a"

	// OUConsumer for consumers
	ClientOUConsumer = "consumer"

	// OUIoTDevice for IoT devices
	ClientOUIoTDevice = "device"

	// OUService for Hiveot services.
	ClientOUService = "service"
)

// CreateCACert creates a CA certificate for signing certificates and digital signatures.
// Intended for self-signed server, client certificates and message signing.
// Source: https://shaneutt.com/blog/golang-ca-and-signed-cert-go/
func CreateCACert(
	cn, country, province, locality, orgName string, validity time.Duration,
	caPrivKey crypto.PrivateKey, caPubKey crypto.PublicKey) (
	caCert *x509.Certificate, err error) {

	// set up our CA certificate
	// see also: https://superuser.com/questions/738612/openssl-ca-keyusage-extension
	// firefox complains if serial is the same as that of the CA. So generate a unique one based on timestamp.
	serial := time.Now().Unix() - 1 // prevent duplicate timestamp with server cert
	rootTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(serial),
		Subject: pkix.Name{
			Country:      []string{country},
			Province:     []string{province},
			Locality:     []string{locality},
			Organization: []string{orgName},
			CommonName:   cn,
		},
		NotBefore: time.Now().Add(-3 * time.Second),
		NotAfter:  time.Now().Add(validity),
		// CA cert can be used to sign certificate and revocation lists
		KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature | x509.KeyUsageCRLSign | x509.KeyUsageDataEncipherment | x509.KeyUsageKeyEncipherment,

		// firefox (2024) seems to consider a CA invalid if extended key usage is
		// combined with regular (critical) key usage???
		// certificate.Verify however fails if ext key usage is just the OCSPSigning.
		//ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageOCSPSigning},
		//ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageOCSPSigning},
		// https://github.com/hashicorp/vault/issues/846 suggests no ext key usage for CA's
		ExtKeyUsage: []x509.ExtKeyUsage{},

		// This hub cert is the only CA. Not using intermediate CAs
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            0,
		MaxPathLenZero:        true,
	}

	// create the self-signed CA certificate
	caCertDer, err := x509.CreateCertificate(
		rand.Reader, rootTemplate, rootTemplate, caPubKey, caPrivKey)
	if err != nil {
		// normally this never happens
		slog.Error("unable to create CA cert", "err", err)
	}
	caCert, err = x509.ParseCertificate(caCertDer)
	return caCert, err
}

// Create a client certificate signed by the given intermediary or CA cert
// intended for testing, not for production
//
//	clientID is the certificate common name, usually the clientID
//	ou the organization the client belongs to
//	country, province, locality, orgName are optional to include
//	validity of the client cert. Required.
//	pubKey is the client's public key for this certificate
//	caCert, caKey are the signing CA's certificate and private key
func CreateClientCert(clientID string, ou string,
	country, province, locality, orgName string,
	validity time.Duration,
	pubKey crypto.PublicKey, caCert *x509.Certificate, caKey crypto.PrivateKey) (
	x509Cert *x509.Certificate, err error) {

	extkeyUsage := x509.ExtKeyUsageClientAuth
	keyUsage := x509.KeyUsageDigitalSignature
	serial := time.Now().Unix() - 2

	template := &x509.Certificate{
		SerialNumber: big.NewInt(serial),
		Subject: pkix.Name{
			Country:            []string{country},
			Province:           []string{province},
			Locality:           []string{locality},
			Organization:       []string{orgName},
			OrganizationalUnit: []string{ou},
			CommonName:         clientID,
			Names:              make([]pkix.AttributeTypeAndValue, 0),
		},
		NotBefore:   time.Now().Add(-10 * time.Second),
		NotAfter:    time.Now().Add(validity),
		KeyUsage:    keyUsage,
		ExtKeyUsage: []x509.ExtKeyUsage{extkeyUsage},

		BasicConstraintsValid: true,
		IsCA:                  false,
	}

	certDerBytes, err := x509.CreateCertificate(
		rand.Reader, template, caCert, pubKey, caKey)
	if err != nil {
		return nil, err
	}
	x509Cert, err = x509.ParseCertificate(certDerBytes)
	return x509Cert, err
}

// Create a server certificate signed by the given intermediary or CA cert
//
// The provided x509 certificate can be converted to a PEM text with:
//
//	certPEM = certs.X509CertToPEM(cert)
//
// To create a TLS cert:
//
//		tlsCert = X509CertToTLS(x509Cert, privKey)
//
//
//	cn is the server common name
//	ou is the organizational unit of the certificate
//	country, province, locality, orgName are optional to include
//
//	names are the SAN names to include with the certificate, localhost and 127.0.0.1 are always added
//	validity is the duration the cert is valid for. Required.
//	serverPubKey contains the server's public key to include in the certificate
//	caCert is the CA certificate used to sign the certificate
//	caKey is the CA private key used to sign certificate
func CreateServerCert(
	cn string, ou string,
	country, province, locality, orgName string,
	names []string,
	validity time.Duration,
	serverPubKey crypto.PublicKey,
	caCert *x509.Certificate, caPrivKey crypto.PrivateKey,
) (x509Cert *x509.Certificate, err error) {

	if cn == "" || serverPubKey == nil {
		err := fmt.Errorf("missing argument cn or serverPubKey")
		slog.Error(err.Error())
		return nil, err
	} else if caCert == nil || caPrivKey == nil {
		err := fmt.Errorf("missing CA certificate or key")
		slog.Error(err.Error())
		return nil, err
	}
	if names == nil {
		names = []string{}
	}
	names = append(names, "127.0.0.1")
	names = append(names, "localhost")

	// firefox complains if serial is the same as that of the CA. So generate a unique one based on timestamp.
	// serial := time.Now().Unix() - 3
	template := x509.Certificate{
		PublicKey:    serverPubKey,
		SerialNumber: nil,
		Subject: pkix.Name{
			Country:            []string{country},
			Province:           []string{province},
			Locality:           []string{locality},
			Organization:       []string{orgName},
			OrganizationalUnit: []string{ou},
			CommonName:         cn,
		},
		NotBefore: time.Now().Add(-time.Second),
		NotAfter:  time.Now().Add(validity),

		KeyUsage: x509.KeyUsageDigitalSignature | x509.KeyUsageDataEncipherment | x509.KeyUsageKeyEncipherment,
		// allow use as both server and client cert
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},

		IsCA:           false,
		MaxPathLenZero: true,
		// BasicConstraintsValid: true,
		// IPAddresses: []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
		IPAddresses: []net.IP{},
	}
	// determine the hosts for this server
	for _, h := range names {
		if ip := net.ParseIP(h); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			template.DNSNames = append(template.DNSNames, h)
		}
	}
	certDerBytes, err := x509.CreateCertificate(
		rand.Reader, &template, caCert, serverPubKey, caPrivKey)
	if err != nil {
		return nil, err
	}
	x509Cert, err = x509.ParseCertificate(certDerBytes)
	return x509Cert, err
}

// Create a new certificate from the given template.
// This uses the template PublicKey in the new certificate
func CreateCertFromTemplate(template *x509.Certificate, validity time.Duration,
	caCert *x509.Certificate, caPrivKey crypto.PrivateKey) (*x509.Certificate, error) {

	certDerBytes, err := x509.CreateCertificate(
		rand.Reader, template, caCert, template.PublicKey, caPrivKey)
	if err != nil {
		return nil, err
	}
	x509Cert, err := x509.ParseCertificate(certDerBytes)
	return x509Cert, err

}

// GetPublicKeyFromCert extracts the public key from x509 certificate.
// Returns nil if certificate doesn't hold a public key.
// The key can be an ecdsa or ed25519 public key.
func GetPublicKeyFromCert(cert *x509.Certificate) (keyType KeyType, pubKey crypto.PublicKey) {

	switch pub := cert.PublicKey.(type) {
	case *ecdsa.PublicKey:
		keyType = KeyTypeECDSA
		pubKey = pub
	case ed25519.PublicKey: // yes, not a pointer
		keyType = KeyTypeED25519
		pubKey = pub
	case *rsa.PublicKey:
		keyType = KeyTypeRSA
		pubKey = pub
	}
	return keyType, pubKey
}

// Load a saved CA certificate from file
// if caKeyPath then ignore the private key
// This returns an error if no valid certificate or key is found.
func LoadCACert(caCertPath string) (caCert *x509.Certificate, err error) {

	chain, err := LoadX509Cert(caCertPath)
	if err != nil {
		// On first start there might not be a CA. Not a fatal error.
		err = fmt.Errorf("No valid CA found at '%s': %w", caCertPath, err)
		return nil, err
	}
	caCert = chain[0]
	return caCert, err
}

// LoadX509Cert loads the x509 certificate chain from a PEM file format.
//
// Intended to load the CA and server certificates.
//
//	pemPath is the full path to the X509 PEM file.
func LoadX509Cert(pemPath string) (cert []*x509.Certificate, err error) {
	pemEncoded, err := os.ReadFile(pemPath)
	if err != nil {
		return nil, err
	}
	return X509ChainFromPEM(string(pemEncoded))
}

// LoadTLSCert loads the TLS certificate from PEM formatted files.
// TLS certificates are a container for both X509 certificate and private key.
//
// Intended to load the certificate and key for servers, or for clients such as IoT devices
// that use client certificate authentication. The idprov service issues this type of
// certificate during IoT device provisioning.
//
// This is simply a wrapper around tls.LoadX509KeyPair. See also SaveTLSCertToPEM.
//
// If loading fails, this returns nil as certificate pointer
func LoadTLSCert(certPEMPath, keyPEMPath string) (tlsCert *tls.Certificate, err error) {
	// golang tls module does it for us
	cert, err := tls.LoadX509KeyPair(certPEMPath, keyPEMPath)
	if err != nil {
		return nil, err
	}
	return &cert, err
}

// SaveTLSCert creates and saves the TLS certificate to a x509 and key file in PEM format.
//
// If the TLS certificate contains a chain then all certificates in the chain are included
// in the cert file.
//
// Intended for saving a certificate received from provisioning or created for testing.
// If the directory doesn't exist it will be created with permissions 755
// The certificate file will be written with permissions 0444. Existing file will be removed first.
// The key file, if provided, will be written with permissions 0400
//
//	tlsCert is the obtained TLS certificate whose parts to save
//	certPemPath the file to save the X509 certificate to in PEM format
//	keyPemPath the file to save the private key to in PEM format. "" to not save the key.
func SaveTLSCert(tlsCert *tls.Certificate, certPemPath, keyPemPath string) error {

	certPem, keyPem := TLSCertToPEM(tlsCert)

	// remove existing cert since perm 0444 doesn't allow overwriting it
	_ = os.Remove(certPemPath)
	_ = os.MkdirAll(filepath.Dir(certPemPath), 0750)

	err := os.WriteFile(certPemPath, []byte(certPem), 0444)
	if err != nil {
		err = fmt.Errorf("Failed writing TLS cert to '%s': %w", certPemPath, err)
		return err
	}

	if keyPemPath == "" {
		return nil
	}
	_ = os.Remove(keyPemPath)
	_ = os.MkdirAll(filepath.Dir(keyPemPath), 0750)

	err = os.WriteFile(keyPemPath, []byte(keyPem), 0400)
	return err
}

// SaveX509Cert saves the x509 certificate to file in PEM format.
//
// If the directory doesn't exist it will be created with permissions 755
// The certificate file will be written with permissions 0444. Existing file will be removed first.
//
// Clients that receive a client certificate from provisioning can use this
// to save the provided certificate to file.
// If the file exists it is removed first.
func SaveX509Cert(cert *x509.Certificate, pemPath string) error {
	certPEM := X509CertToPEM(cert)

	// remove existing cert since perm 0444 doesn't allow overwriting it
	_ = os.Remove(pemPath)
	_ = os.MkdirAll(filepath.Dir(pemPath), 0750)
	err := os.WriteFile(pemPath, []byte(certPEM), 0444)
	return err
}

// SaveX509CertChain saves the x509 certificate chain to file in PEM format.
// If the file exists it is removed first.
func SaveX509CertChain(certChain []*x509.Certificate, pemPath string) error {

	certPEM := X509ChainToPEM(certChain)

	// remove existing cert since perm 0444 doesn't allow overwriting it
	_ = os.Remove(pemPath)
	_ = os.MkdirAll(filepath.Dir(pemPath), 0750)
	err := os.WriteFile(pemPath, []byte(certPEM), 0444)
	return err
}

// TLSCertToX509 splits a TLS certificate into an x509 certificate chain and private key
// If the TLS cert does not contain a private key it is returned as nil.
//
// This returns nil if the tls certificate doesn't hold any x509 certificates
func TLSCertToX509(tlsCert *tls.Certificate) ([]*x509.Certificate, crypto.PrivateKey) {
	certChain := make([]*x509.Certificate, 0)
	// A TLS certificate is a wrapper around x509 with private key
	for _, rawCert := range tlsCert.Certificate {
		cert, err := x509.ParseCertificate(rawCert)
		if err != nil {
			return nil, nil
		}
		certChain = append(certChain, cert)
	}
	// if tlsCert.PrivateKey == nil {
	// 	err = fmt.Errorf("missing private key")
	// }

	return certChain, tlsCert.PrivateKey
}

// TLSCertFromPEM converts a PEM text to a TLS certificate.
//
// This support a concatenated PEM text.
// If the provided PEM only contains a certificate then the TLS wont have a
// private key part.
func TLSCertFromPEM(tlsPem string) (*tls.Certificate, error) {

	// X509KeyPair just picks the compatible block from the PEM
	tlsCert, err := tls.X509KeyPair([]byte(tlsPem), []byte(tlsPem))
	return &tlsCert, err
}

// TLSCertToPEM converts a TLS certificate to a certificate and a private key PEM string.
// If the TLS cert does not contain a private key it is returned as empty.
func TLSCertToPEM(tlsCert *tls.Certificate) (certPem string, keyPem string) {

	x509Chain, privKey := TLSCertToX509(tlsCert)

	certPem = X509ChainToPEM(x509Chain)
	keyPem = PrivateKeyToPEM(privKey)
	return certPem, keyPem
}

// VerifyCert verifies whether the given certificate is a valid certificate for client authentication
// This returns the certificate CN as the clientID
func VerifyCert(cert *x509.Certificate, caCertPool *x509.CertPool) (cn string, err error) {

	opts := x509.VerifyOptions{
		Roots:     caCertPool,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	if cert.Subject.CommonName == "" {
		err = fmt.Errorf("cert has no CommonName")
	}

	//if err == nil {
	//	x509Cert, err := x509.ParseCertificate(clientCert.Certificate[0])
	//}
	if err == nil {
		// FIXME: TestCertAuth: certificate specifies incompatible key usage
		// why? Is the certpool invalid? Yet the test succeeds
		_, err = cert.Verify(opts)
	}
	return cert.Subject.CommonName, err
}

// X509CertFromPEM converts the PEM format certificate chain to a X509 certificate
// If the pem contains a chain only the first is returned.
func X509CertFromPEM(certPem string) (cert *x509.Certificate, err error) {
	chain, err := X509ChainFromPEM(certPem)
	if err != nil {
		return nil, err
	}
	return chain[0], err
}

// X509ChainFromPEM converts the PEM format certificate chain to a X509 certificate chain
func X509ChainFromPEM(certPem string) (chain []*x509.Certificate, err error) {

	chain = make([]*x509.Certificate, 0)

	// certChain, err := x509.ParseCertificates([]byte(certPem))

	// first cert
	pemBlock, remainder := pem.Decode([]byte(certPem))
	if pemBlock == nil {
		return nil, fmt.Errorf("X509CertFromPEM: Provided certPem is not a valid PEM encoding")
	}

	// additional certificates from the chain
	for pemBlock != nil {
		x509Cert, err := x509.ParseCertificate(pemBlock.Bytes)
		if err != nil {
			return nil, fmt.Errorf("X509CertFromPEM: x509 ParseCertificate failed: %w", err)
		}
		chain = append(chain, x509Cert)
		// get the next block
		pemBlock, remainder = pem.Decode(remainder)
	}
	return chain, err
}

// X509ChainToPEM converts the x509 certificate chain to PEM format
func X509ChainToPEM(chain []*x509.Certificate) (certStr string) {
	certStr = ""
	for _, cert := range chain {
		b := pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw}
		certPEM := pem.EncodeToMemory(&b)
		certStr = certStr + string(certPEM)
	}
	return string(certStr)
}

// X509CertToPEM converts the x509 certificate to PEM format
func X509CertToPEM(cert *x509.Certificate) (certPem string) {
	b := pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw}
	certPEM := pem.EncodeToMemory(&b)
	return string(certPEM)
}

// X509CertToTLS creates a TLS certificate from a x509 certificate and private key
func X509CertToTLS(cert *x509.Certificate, privKey crypto.PrivateKey) *tls.Certificate {
	chain := []*x509.Certificate{cert}
	return X509ChainToTLS(chain, privKey)
}

// X509ChainToTLS combines a x509 certificate chain and private key into a TLS certificate
func X509ChainToTLS(chain []*x509.Certificate, privKey crypto.PrivateKey) *tls.Certificate {
	// A TLS certificate is a wrapper around x509 with private key
	tlsCert := tls.Certificate{}
	for _, x509 := range chain {
		tlsCert.Certificate = append(tlsCert.Certificate, x509.Raw)
	}
	tlsCert.PrivateKey = privKey

	return &tlsCert
}

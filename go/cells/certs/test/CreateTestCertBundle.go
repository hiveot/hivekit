// Package certs with managing certificates for testing
package certstest

import (
	"crypto"
	"crypto/tls"
	"crypto/x509"
	"time"

	"github.com/hiveot/hivekit/go/utils"
)

// can't use localhost because discovery needs the outbound interface
var ServerAddress = utils.GetOutboundIP("").String() //"127.0.0.1"
const TestServerID = "server1"
const TestClientID = "client1"

// TestCertBundle creates a set of CA, server and client certificates intended for testing
type TestCertBundle struct {
	// the key type used to generate the private keys
	keyType utils.KeyType

	// CA
	RootCAs   *x509.CertPool
	CaCert    *x509.Certificate
	CaPrivKey crypto.PrivateKey
	CaPubKey  crypto.PublicKey

	// server certificate
	ServerAddr    string
	ServerPrivKey crypto.PrivateKey
	ServerPubKey  crypto.PublicKey
	ServerCert    *tls.Certificate

	// client cert auth
	ClientID      string
	ClientPrivKey crypto.PrivateKey
	ClientPubKey  crypto.PublicKey
	ClientCert    *tls.Certificate
}

// CreateTestCertBundle creates a bundle of ca, server certificates and keys for testing.
// The server cert is valid for the 127.0.0.1, localhost, os.hostname and outbound IP.
func CreateTestCertBundle(keyType utils.KeyType) TestCertBundle {
	var err error
	// cfg := certs.CertsConfig{}
	// provider := selfsignedimpl.NewSelfSignedProvider(&cfg)

	certBundle := TestCertBundle{
		keyType:    keyType,
		ServerAddr: ServerAddress,
	}
	// Setup CA and server TLS certificates
	caPrivKey, caPubKey := utils.NewKey(keyType)
	caCert, err := utils.CreateCACert(
		"testbundleca", "country", "province", "locality", "orgName",
		time.Hour, caPrivKey, caPubKey)
	certBundle.CaCert = caCert
	certBundle.CaPrivKey = caPrivKey
	certBundle.CaPubKey = caPubKey

	if err != nil {
		panic("CreateCertBundler failed: " + err.Error())
	}
	certBundle.RootCAs, _ = x509.SystemCertPool()
	certBundle.RootCAs.AddCert(certBundle.CaCert)
	certBundle.ServerPrivKey, certBundle.ServerPubKey = utils.NewKey(keyType)
	certBundle.ClientPrivKey, certBundle.ClientPubKey = utils.NewKey(keyType)

	names := []string{ServerAddress}
	serverCert, err := utils.CreateServerCert(
		TestServerID, "ou", "country", "province", "locality", "org", names, time.Hour,
		certBundle.ServerPubKey,
		certBundle.CaCert, certBundle.CaPrivKey)
	if err != nil {
		panic("unable to create server cert: " + err.Error())
	}
	// certBundle.ServerCert = X509CertToTLS(serverCert, certBundle.ServerKey)
	certBundle.ServerCert = &tls.Certificate{
		Certificate: [][]byte{serverCert.Raw},
		PrivateKey:  certBundle.ServerPrivKey,
	}
	certBundle.ClientID = TestClientID
	clientCert, err := utils.CreateClientCert(certBundle.ClientID, "service",
		"country", "province", "locality", "org", time.Hour,
		certBundle.ClientPubKey,
		certBundle.CaCert,
		certBundle.CaPrivKey)
	if err != nil {
		panic("unable to create client cert: " + err.Error())
	}
	// certBundle.ClientCert = X509CertToTLS(clientCert, certBundle.ClientKey)
	certBundle.ClientCert = &tls.Certificate{
		Certificate: [][]byte{clientCert.Raw},
		PrivateKey:  certBundle.ClientPrivKey,
	}

	return certBundle
}

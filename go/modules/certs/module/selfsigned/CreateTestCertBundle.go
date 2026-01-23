// Package certs with managing certificates for testing
package selfsigned

import (
	"crypto"
	"crypto/tls"
	"crypto/x509"

	"github.com/hiveot/hivekit/go/utils"
)

const ServerAddress = "127.0.0.1"
const TestServerID = "server1"
const TestClientID = "client1"

// TestCertBundle creates a set of CA, server and client certificates intended for testing
type TestCertBundle struct {
	// the key type used to generate the private keys
	keyType utils.KeyType

	// CA
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
// The server cert is valid for the 127.0.0.1, localhost and os.hostname.
func CreateTestCertBundle(keyType utils.KeyType) TestCertBundle {
	var err error
	certBundle := TestCertBundle{
		keyType:    keyType,
		ServerAddr: ServerAddress,
	}
	// Setup CA and server TLS certificates
	certBundle.CaCert, certBundle.CaPrivKey, certBundle.CaPubKey, err =
		CreateSelfSignedCA("", "", "", "", "testbundleca", 1, keyType)
	if err != nil {
		panic("CreateCertBundler failed: " + err.Error())
	}
	certBundle.ServerPrivKey, certBundle.ServerPubKey = utils.NewKey(keyType)
	certBundle.ClientPrivKey, certBundle.ClientPubKey = utils.NewKey(keyType)

	names := []string{ServerAddress, "localhost"}
	serverCert, err := CreateSelfSignedServerCert(
		TestServerID, "server", 1,
		certBundle.ServerPubKey,
		names,
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
	clientCert, err := CreateClientCert(certBundle.ClientID, "service", 1,
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

package utils_test

import (
	"crypto/x509"
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"

	"github.com/hiveot/hivekit/go/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const TestKeyType = utils.KeyTypeED25519

var storageDir = filepath.Join(os.TempDir(), "certs")

func TestCreateCA(t *testing.T) {

	// ceate CA cert
	caPrivKey, caPubKey := utils.NewKey(TestKeyType)
	require.NotEmpty(t, caPubKey)
	caCert, err := utils.CreateCACert("testCA", "country", "province", "locality", "orgName", time.Hour, caPrivKey, caPubKey)
	require.NoError(t, err)
	require.NotEmpty(t, caCert)

}

// Create and verify the server and client cert against the CA
func TestCreateVerifyCerts(t *testing.T) {
	// test creating hub certificate
	const serverID = "testService"
	const clientID = "testClient"
	names := []string{"127.0.0.1", "localhost"}

	caPrivKey, caPubKey := utils.NewKey(TestKeyType)
	caCert, err := utils.CreateCACert("testCA", "country", "province", "locality", "orgName", time.Hour, caPrivKey, caPubKey)
	require.NoError(t, err)
	require.NotEmpty(t, caCert)
	caPool := x509.NewCertPool()
	caPool.AddCert(caCert)

	// verify a server cert
	serverPrivateKey, serverPublicKey := utils.NewKey(TestKeyType)
	_ = serverPrivateKey
	serverCert, err := utils.CreateServerCert(
		serverID, "myou", "country", "province", "locality", "org",
		names, time.Hour,
		serverPublicKey, caCert, caPrivKey)

	cn, err := utils.VerifyCert(serverCert, caPool)

	assert.NoError(t, err)
	assert.Equal(t, serverID, cn)

	// verify a client cert
	clientPrivateKey, clientPublicKey := utils.NewKey(TestKeyType)
	_ = clientPrivateKey
	clientCert, err := utils.CreateClientCert(
		clientID, "myou", "country", "province", "locality", "org",
		time.Hour,
		clientPublicKey, caCert, caPrivKey)
	assert.NoError(t, err)
	assert.NotEmpty(t, clientCert)

	cn, err = utils.VerifyCert(clientCert, caPool)
	assert.NoError(t, err)
	assert.Equal(t, clientID, cn)

}

func TestPublicKeyFromCert(t *testing.T) {
	caPrivKey, caPubKey := utils.NewKey(keyType)
	caCert, _ := utils.CreateCACert(
		"testbundleca", "country", "province", "locality", "orgName",
		time.Hour, caPrivKey, caPubKey)

	certKeyType, certPubKey := utils.GetPublicKeyFromCert(caCert)
	assert.Equal(t, caPubKey, certPubKey)
	assert.Equal(t, keyType, certKeyType)
}

// create a CA and server cert, save it and reload it
func TestSaveLoadTLSCert(t *testing.T) {
	caPool := x509.NewCertPool()
	caPrivKey, caPubKey := utils.NewKey(keyType)
	caCert, err := utils.CreateCACert(
		"testbundleca", "country", "province", "locality", "orgName",
		time.Hour, caPrivKey, caPubKey)
	caPool.AddCert(caCert)

	serverPrivKey, serverPubKey := utils.NewKey(keyType)
	_ = serverPrivKey
	serverCert, err := utils.CreateServerCert(
		"serverID", "ou", "country", "province", "locality", "org", nil, time.Hour,
		serverPubKey, caCert, caPrivKey)
	serverChain := []*x509.Certificate{serverCert, caCert}

	// save the test server cert
	certFile := path.Join(storageDir, "serverCert.pem")
	err = utils.SaveX509CertChain(serverChain, certFile)
	assert.NoError(t, err)

	serverCert2, err := utils.LoadX509Cert(certFile)
	require.NoError(t, err)
	require.NotEmpty(t, serverCert2)

	// after loading the cert must still be valid
	cn, err := utils.VerifyCert(serverCert2[0], caPool)
	assert.NoError(t, err)
	assert.NotEmpty(t, cn)

}

// test with bad parameters
func TestServerCertBadParms(t *testing.T) {
	const serverID = "testService"
	names := []string{"127.0.0.1", "localhost"}

	caPrivKey, caPubKey := utils.NewKey(TestKeyType)
	caCert, err := utils.CreateCACert("testCA", "country", "province", "locality", "orgName",
		time.Hour, caPrivKey, caPubKey)

	serverKey, _ := utils.NewEcdsaKey()

	// Missing CA certificate
	_, err = utils.CreateServerCert(
		serverID, "myou", "country", "province", "locality", "orgName",
		names, time.Hour, &serverKey.PublicKey, nil, caPrivKey)
	assert.Error(t, err)

	// missing CA private key
	_, err = utils.CreateServerCert(
		serverID, "myou", "country", "province", "locality", "orgName",
		names, time.Hour, &serverKey.PublicKey, caCert, nil)
	assert.Error(t, err)

	// missing service ID
	serverCert, err := utils.CreateServerCert(
		"", "myou", "country", "province", "locality", "orgName",
		names, time.Hour, &serverKey.PublicKey, caCert, caPrivKey)
	_ = serverCert
	require.Error(t, err)
	require.Empty(t, serverCert)

	// missing public key
	serverCert, err = utils.CreateServerCert(
		serverID, "myou", "country", "province", "locality", "orgName",
		names, time.Hour, nil, caCert, caPrivKey)
	require.Error(t, err)
	require.Empty(t, serverCert)

}

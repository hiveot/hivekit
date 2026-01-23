package selfsigned_test

import (
	"encoding/pem"
	"testing"

	"github.com/hiveot/hivekit/go/modules/certs/module/selfsigned"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const TestKeyType = utils.KeyTypeED25519

func TestCreateCerts(t *testing.T) {
	// test creating hub certificate
	const serverID = "testService"
	const clientID = "testClient"
	names := []string{"127.0.0.1", "localhost"}

	caCert, caPrivKey, caPubKey, err := selfsigned.CreateSelfSignedCA(
		"", "", "", "", "testca", 1, TestKeyType)
	require.NoError(t, err)
	require.NotEmpty(t, caPubKey)

	serverPrivateKey, serverPublicKey := utils.NewKey(TestKeyType)
	_ = serverPrivateKey
	serverCert, err := selfsigned.CreateSelfSignedServerCert(
		serverID, "myou", 0, serverPublicKey, names, caCert, caPrivKey)
	assert.NoError(t, err)
	assert.NotEmpty(t, serverCert)
	// cert to pem
	b := pem.Block{Type: "CERTIFICATE", Bytes: serverCert.Raw}
	certPem := string(pem.EncodeToMemory(&b))

	// verify the created cert
	cn, err := selfsigned.VerifyPemCert(certPem, caCert)
	assert.NoError(t, err)
	assert.Equal(t, serverID, cn)

	clientPrivateKey, clientPublicKey := utils.NewKey(TestKeyType)
	_ = clientPrivateKey
	clientCert, err := selfsigned.CreateClientCert(
		clientID, "", 0, clientPublicKey, caCert, caPrivKey)
	assert.NoError(t, err)
	assert.NotEmpty(t, clientCert)
}

// test with bad parameters
func TestServerCertBadParms(t *testing.T) {
	const serverID = "testService"
	names := []string{"127.0.0.1", "localhost"}

	caCert, caPrivKey, _, _ := selfsigned.CreateSelfSignedCA(
		"", "", "", "", "testca", 1, TestKeyType)
	serverKey, _ := utils.NewEcdsaKey()

	// Missing CA certificate
	_, err := selfsigned.CreateSelfSignedServerCert(
		serverID, "myou", 0, &serverKey.PublicKey, names, nil, caPrivKey)
	assert.Error(t, err)

	// missing CA private key
	_, err = selfsigned.CreateSelfSignedServerCert(
		serverID, "myou", 0, &serverKey.PublicKey, names, caCert, nil)
	assert.Error(t, err)

	// missing service ID
	serverCert, err := selfsigned.CreateSelfSignedServerCert(
		"", "myou", 0, &serverKey.PublicKey, names, caCert, caPrivKey)
	_ = serverCert
	require.Error(t, err)
	require.Empty(t, serverCert)

	// missing public key
	serverCert, err = selfsigned.CreateSelfSignedServerCert(
		serverID, "myou", 0, nil, names, caCert, caPrivKey)
	require.Error(t, err)
	require.Empty(t, serverCert)

}

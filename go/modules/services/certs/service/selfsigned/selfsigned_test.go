package selfsigned_test

import (
	"encoding/pem"
	"testing"

	"github.com/hiveot/hivekit/go/modules/services/certs/keys"
	"github.com/hiveot/hivekit/go/modules/services/certs/service/selfsigned"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateCerts(t *testing.T) {
	// test creating hub certificate
	const serverID = "testService"
	const clientID = "testClient"
	names := []string{"127.0.0.1", "localhost"}

	caCert, caKeyPair, err := selfsigned.CreateCA("", "", "", "", "testca", 1)
	require.NoError(t, err)

	serverKey := keys.NewEcdsaKey()
	serverCert, err := selfsigned.CreateServerCert(
		serverID, "myou", 0, serverKey, names, caCert, caKeyPair)
	assert.NoError(t, err)
	assert.NotEmpty(t, serverCert)
	// cert to pem
	b := pem.Block{Type: "CERTIFICATE", Bytes: serverCert.Raw}
	certPem := string(pem.EncodeToMemory(&b))

	// verify the created cert
	cn, err := selfsigned.VerifyPemCert(certPem, caCert)
	assert.NoError(t, err)
	assert.Equal(t, serverID, cn)

	// serverCertPEM := service.X509CertToPEM(serverCert)
	// // verify service certificate against CA
	// _, err = selfsigned.VerifyCert(serverCertPEM, caCert)
	// assert.NoError(t, err)

	// create a server TLS cert
	// tlsCert := selfsigned.X509CertToTLS(serverCert, serverKey)
	// assert.NotEmpty(t, tlsCert)

	// create a client cert
	clientKey := keys.NewKey(keys.KeyTypeEd25519)
	clientCert, err := selfsigned.CreateClientCert(
		clientID, "", 0, clientKey, caCert, caKeyPair)
	assert.NoError(t, err)
	assert.NotEmpty(t, clientCert)
}

// test with bad parameters
func TestServerCertBadParms(t *testing.T) {
	const serverID = "testService"
	names := []string{"127.0.0.1", "localhost"}

	caCert, caKey, _ := selfsigned.CreateCA("", "", "", "", "testca", 1)
	serverKey := keys.NewKey(keys.KeyTypeECDSA)

	// Missing CA certificate
	_, err := selfsigned.CreateServerCert(
		serverID, "myou", 0, serverKey, names, nil, caKey)
	assert.Error(t, err)

	// missing CA private key
	_, err = selfsigned.CreateServerCert(
		serverID, "myou", 0, serverKey, names, caCert, nil)
	assert.Error(t, err)

	// missing service ID
	serverCert, err := selfsigned.CreateServerCert(
		"", "myou", 0, serverKey, names, caCert, caKey)
	_ = serverCert
	require.Error(t, err)
	require.Empty(t, serverCert)

	// missing public key
	serverCert, err = selfsigned.CreateServerCert(
		serverID, "myou", 0, nil, names, caCert, caKey)
	require.Error(t, err)
	require.Empty(t, serverCert)

}

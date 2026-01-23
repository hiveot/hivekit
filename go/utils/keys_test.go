package utils_test

import (
	"crypto/ecdsa"
	"os"
	"path"
	"testing"

	"github.com/hiveot/hivekit/go/lib/logging"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var keyType utils.KeyType = utils.KeyTypeED25519

// set in TestMain
var TestKeysFolder string
var testPrivKeyPemFile string
var testPubKeyPemFile string

// TestMain create a test folder for keys
func TestMain(m *testing.M) {
	TestKeysFolder, _ = os.MkdirTemp("", "hiveot-keys-")

	testPrivKeyPemFile = path.Join(TestKeysFolder, "privKey.pem")
	testPubKeyPemFile = path.Join(TestKeysFolder, "pubKey.pem")
	logging.SetLogging("info", "")

	result := m.Run()
	if result != 0 {
		println("Test failed with code:", result)
		println("Find test files in:", TestKeysFolder)
	} else {
		// comment out the next line to be able to inspect results
		_ = os.RemoveAll(TestKeysFolder)
	}

	os.Exit(result)
}

func TestMultipleKeys(t *testing.T) {
	var keyTypes = []utils.KeyType{utils.KeyTypeECDSA, utils.KeyTypeRSA, utils.KeyTypeED25519}
	for i := 0; i < len(keyTypes); i++ {
		keyType = keyTypes[i]
		t.Logf("\n\n---%s---; keyType=%s\n", t.Name(), keyType)
		t.Run("TestSaveLoadPrivKey", TestSaveLoadPrivKey)
		t.Run("TestSaveLoadPubkey", TestSaveLoadPubkey)
		t.Run("TestSaveLoadPrivKeyNotFound", TestSaveLoadPrivKeyNotFound)
		t.Run("TestSaveLoadPubKeyNotFound", TestSaveLoadPubKeyNotFound)
		t.Run("TestPublicKeyPEM", TestPublicKeyPEM)
		t.Run("TestPrivateKeyPEM", TestPrivateKeyPEM)
		t.Run("TestInvalidEnc", TestInvalidEnc)

	}
}

func TestSaveLoadPrivKey(t *testing.T) {
	t.Logf("---%s---\n", t.Name())

	k1priv, k1pub := utils.NewKey(keyType)
	err := utils.SavePrivateKey(k1priv, testPrivKeyPemFile)
	assert.NoError(t, err)

	k2KeyType, k2priv, k2pub, err := utils.LoadPrivateKey(testPrivKeyPemFile)
	assert.NoError(t, err)
	require.NotNil(t, k2priv)
	require.Equal(t, keyType, k2KeyType)

	pem1 := utils.PublicKeyToPem(k1pub)
	pem2 := utils.PublicKeyToPem(k2pub)
	assert.NotEmpty(t, pem1)
	assert.Equal(t, pem1, pem2)

	msg := []byte("hello world")
	signature, err := utils.Sign(msg, k1priv)
	require.NoError(t, err)
	valid := utils.Verify(msg, signature, k2pub)
	assert.True(t, valid)

	//
	// privKey, pubKey, err := utils.LoadCreateKeyPair(clientID, keysDir, keyType)
	assert.NoError(t, err)
}

func TestSaveLoadPubkey(t *testing.T) {
	t.Logf("---%s---\n", t.Name())

	k1, k1Pub := utils.NewKey(keyType)
	require.NotEmpty(t, k1)
	err := utils.SavePublicKey(k1Pub, testPubKeyPemFile)
	assert.NoError(t, err)

	k2Pub, err := utils.LoadPublicKey(testPubKeyPemFile)
	require.NoError(t, err)
	require.NotEmpty(t, k2Pub)
	pubEnc := utils.PublicKeyToPem(k2Pub)
	assert.NotEmpty(t, pubEnc)
}

func TestSaveLoadPrivKeyNotFound(t *testing.T) {
	t.Logf("---%s---\n", t.Name())

	k1, _ := utils.NewKey(keyType)
	// no access
	err := utils.SavePrivateKey(k1, "/root")
	assert.Error(t, err)

	//
	k1, _, _, err = utils.LoadPrivateKey("/filedoesnotexist.pem")
	assert.Error(t, err)
}

func TestSaveLoadPubKeyNotFound(t *testing.T) {
	t.Logf("---%s---\n", t.Name())

	pubKey, err := utils.LoadPublicKey("/filedoesnotexist.pem")
	assert.Error(t, err)
	assert.Nil(t, pubKey)
}

func TestPublicKeyPEM(t *testing.T) {
	t.Logf("---%s---\n", t.Name())

	// golang generated public key (91 bytes after base64 decode) - THIS WORKS
	const TestKeyPub2 = "MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEFFPcFfnGQr8/t2ZzWFYg/ZFLAkT0z/EYlC1RED4iot367KRNwZlilogTGHzi3HjH6NnL14d/DQHxAInctEeqxw=="
	const TestKeyPubPEM2 = "-----BEGIN PUBLIC KEY-----\nMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEaejQVxAbrUiN41Nqjgw8HG8q5OQM\nkveXku18/zhF2BbSfbQMnCSyP5VXCe/sgCEi62Qm0LYXd1VG2UQz38f4zQ==\n-----END PUBLIC KEY-----\n"

	// JS ellipsys generated public key (65 bytes after base64 decode) - THIS FAILS
	const TestJSEllipsysPub3 = "BKOVp2t2JLjodototsMvFbOJ1j9wTC4ITbOrnrb/EoJiQul9eoXmyHpaYnPztjPixFdiHk06NxGLDpxRDm5qXfo="

	// openssl generated public key - THIS WORKS
	const TestKeyOpenSSL = "MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAELtv253KXEWvWjCse0Wp5DprnXp5tp17C1Qtfjk5t/6+HPSc74uMQcp/KV++vc6OXJwk5XdZ8FSkUiU9cYRBo8A=="

	// JS elliptic encoded using base64 encoding of hex encoded pub key - this fails
	const TestKeyPub64hex = "MDRhMzk1YTc2Yjc2MjRiOGU4NzY4YjY4YjZjMzJmMTViMzg5ZDYzZjcwNGMyZTA4NGRiM2FiOWViNmZmMTI4MjYyNDJlOTdkN2E4NWU2Yzg3YTVhNjI3M2YzYjYzM2UyYzQ1NzYyMWU0ZDNhMzcxMThiMGU5YzUxMGU2ZTZhNWRmYQ=="

	// decode from base64 string. This succeeds. d2 is 91 bytes - this works
	k1, err := utils.PublicKeyFromPem(TestKeyPub2)
	require.NoError(t, err)
	assert.NotNil(t, k1)
	_, valid := k1.(*ecdsa.PublicKey)
	assert.True(t, valid)

	// decode from openssl generated public key. d2 is 91 bytes - this works
	k2, err := utils.PublicKeyFromPem(TestKeyOpenSSL)
	assert.NotNil(t, k2)
	_, valid = k1.(*ecdsa.PublicKey)
	assert.True(t, valid)

	// a hex key is not supported
	k3, err := utils.PublicKeyFromPem(TestKeyPub64hex)
	assert.Nil(t, k3)

	////MarshalPKIXPublicKey converts a public key to PKIX, ASN.1 DER form
	//x509EncodedPub, err := x509.MarshalPKIXPublicKey(publicKey)
	//pemEncodedPub := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: x509EncodedPub})

	// Parse JS elliptic generated key.
	// THIS FAILS as it is hex based without a DER prefix
	// d3 is 65 bytes which should be correct
	k4, err := utils.PublicKeyFromPem(TestJSEllipsysPub3)
	assert.Nil(t, k4)
}

func TestPrivateKeyPEM(t *testing.T) {
	t.Logf("---%s---\n", t.Name())

	k1, _ := utils.NewKey(keyType)
	k1Pem := utils.PrivateKeyToPem(k1)
	assert.NotEmpty(t, k1Pem)

	kt1 := utils.DetermineKeyType(k1Pem)
	assert.Equal(t, keyType, kt1)

	keyType2, k2, _, err := utils.PrivateKeyFromPem(k1Pem)
	require.NoError(t, err)
	assert.Equal(t, keyType, keyType2)

	k2Pem := utils.PrivateKeyToPem(k2)
	require.NotNil(t, k2Pem)

	isEqual := k1Pem == k2Pem
	assert.True(t, isEqual)
}

func TestInvalidEnc(t *testing.T) {
	t.Logf("---%s---\n", t.Name())

	ktype, _, _, err := utils.PrivateKeyFromPem("PRIVATE KEY")
	assert.Equal(t, utils.KeyTypeUnknown, ktype)
	assert.Error(t, err)

	// note: nkeys have not ability to verify the public key
	_, err = utils.PublicKeyFromPem("PUBLIC KEY")
	assert.Error(t, err)
}

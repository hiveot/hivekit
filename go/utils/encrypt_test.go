package utils_test

import (
	"crypto/rand"
	"encoding/base64"
	"testing"

	"github.com/hiveot/hivekit/go/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncryptDecrypt(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	var myprotectedtext = "This is protected text"

	key := make([]byte, 24)
	_, err := rand.Read(key)
	secret := base64.StdEncoding.EncodeToString(key)

	encryptedData, err := utils.Encrypt([]byte(myprotectedtext), []byte(secret))
	require.NoError(t, err)

	decryptedData, err := utils.Decrypt(encryptedData, []byte(secret))
	require.NoError(t, err)

	assert.Equal(t, myprotectedtext, string(decryptedData))

}

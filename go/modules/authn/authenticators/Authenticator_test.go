package authenticators_test

import (
	"os"
	"path"
	"testing"
	"time"

	"github.com/hiveot/hivekit/go/modules/authn"
	"github.com/hiveot/hivekit/go/modules/authn/authenticators"
	"github.com/hiveot/hivekit/go/modules/authn/service/authnstore"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var authnStore authnstore.IAuthnStore
var testDir = path.Join(os.TempDir(), "test-authn")
var defaultHash = authn.PWHASH_ARGON2id

func NewAuthenticator() transports.IAuthenticator {
	passwordFile := path.Join(testDir, "test.passwd")
	authnStore = authnstore.NewAuthnFileStore(passwordFile, defaultHash)

	// signingKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	// svc := authenticator.NewJWTAuthenticator(authnStore, signingKey, "")

	signingPrivKey, _ := utils.NewEd25519Key()
	svc := authenticators.NewPasetoAuthenticator(authnStore, signingPrivKey)
	svc.SetAuthServerURI("/fake/server/endpoint")
	return svc
}

func TestCreateSessionToken(t *testing.T) {
	const clientID = "user1"
	const pass1 = "pass1"
	const role = "role1"
	//const clientType = authn.ClientTypeConsumer

	svc := NewAuthenticator()
	_ = authnStore.Add(authn.ClientProfile{
		ClientID:    clientID,
		Role:        authn.ClientRoleViewer,
		Disabled:    false,
		DisplayName: "test",
	})
	err := authnStore.SetPassword(clientID, pass1)
	require.NoError(t, err)

	token1, validUntil := svc.CreateToken(clientID, role, time.Minute)
	assert.NotEmpty(t, token1)
	assert.Greater(t, validUntil, time.Now())

	// decode it
	clientID2, role2, issuedAt2, validUntil2, err := svc.DecodeToken(token1, "", "")
	require.NoError(t, err)
	assert.Less(t, issuedAt2, time.Now())
	require.Equal(t, clientID, clientID2)
	// require.LessOrEqual(t, validUntil, validUntil2)  // second is truncated
	require.Equal(t, role, role2)

	// logout
	svc.Logout(clientID2)

	// validate the new token. Without a session this fails
	clientID3, role2, validUntil, err := svc.ValidateToken(token1)
	require.Error(t, err)
	require.Equal(t, clientID, clientID3)
	require.Equal(t, role, role2)
	require.Greater(t, validUntil, time.Now())

	_, _, err = svc.Login(clientID, pass1)

	// create a persistent auth token
	token2, validUntil := svc.CreateToken(clientID, role, time.Minute)
	clientID4, role3, validUntil2, err := svc.ValidateToken(token2)
	require.NoError(t, err)
	require.Equal(t, clientID, clientID4)
	require.Equal(t, role, role3)
	require.Equal(t, validUntil.Unix(), validUntil2.Unix())

}

func TestBadTokens(t *testing.T) {
	const clientID = "user1"
	const role = "role1"
	//const clientType = authn.ClientTypeConsumer

	svc := NewAuthenticator()

	token1, validUntil := svc.CreateToken(clientID, role, time.Minute)
	assert.NotEmpty(t, token1)
	assert.Greater(t, validUntil, time.Now())

	// try to refresh as a different client

	// refresh
	badToken := token1 + "-bad"
	_, _, _, err := svc.ValidateToken(badToken)
	require.Error(t, err)

	// expired
	token2, _ := svc.CreateToken(clientID, role, -1)
	clientID2, _, sid2, err := svc.ValidateToken(token2)
	require.Error(t, err)
	assert.Empty(t, clientID2)
	assert.Empty(t, sid2)

}

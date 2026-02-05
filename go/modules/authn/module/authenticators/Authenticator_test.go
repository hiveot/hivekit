package authenticators_test

import (
	"os"
	"path"
	"testing"
	"time"

	"github.com/hiveot/hivekit/go/modules/authn"
	"github.com/hiveot/hivekit/go/modules/authn/module/authenticators"
	"github.com/hiveot/hivekit/go/modules/authn/module/authnstore"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var authnStore authnstore.IAuthnStore
var testDir = path.Join(os.TempDir(), "test-authn")
var defaultHash = authn.PWHASH_ARGON2id

func NewAuthenticator() (transports.IAuthenticator, authnstore.IAuthnStore) {
	passwordFile := path.Join(testDir, "test.passwd")
	authnStore = authnstore.NewAuthnFileStore(passwordFile, defaultHash)

	// signingKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	// svc := authenticator.NewJWTAuthenticator(authnStore, signingKey, "")

	signingPrivKey, _ := utils.NewEd25519Key()
	svc := authenticators.NewPasetoAuthenticator(authnStore, signingPrivKey)
	svc.SetAuthServerURI("/fake/server/endpoint")
	return svc, authnStore
}

func TestCreateSessionToken(t *testing.T) {
	const clientID = "user1"
	const pass1 = "pass1"
	const role = "role1"
	//const clientType = authn.ClientTypeConsumer

	svc, clientStore := NewAuthenticator()
	_ = clientStore.Add(authn.ClientProfile{
		ClientID:    clientID,
		Role:        role,
		Disabled:    false,
		DisplayName: "test",
	})
	err := authnStore.SetPassword(clientID, pass1)
	require.NoError(t, err)

	token1, validUntil, err := svc.CreateToken(clientID, time.Minute)
	require.NoError(t, err)
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
	require.NoError(t, err)

	// create a persistent auth token
	token2, validUntil, err := svc.CreateToken(clientID, time.Minute)
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

	svc, clientStore := NewAuthenticator()
	_ = clientStore.Add(authn.ClientProfile{
		ClientID:    clientID,
		Role:        transports.ClientRoleViewer,
		Disabled:    false,
		DisplayName: "test",
	})

	token1, validUntil, err := svc.CreateToken(clientID, time.Minute)
	assert.NotEmpty(t, token1)
	assert.Greater(t, validUntil, time.Now())

	// bad token
	badToken := token1 + "-bad"
	_, _, _, err = svc.ValidateToken(badToken)
	require.Error(t, err)

	// expired
	token2, _, err := svc.CreateToken(clientID, -1)
	require.NoError(t, err)
	clientID2, _, sid2, err := svc.ValidateToken(token2)
	require.Error(t, err)
	assert.Empty(t, clientID2)
	assert.Empty(t, sid2)

	// missing clientID
	token2, _, err = svc.CreateToken("", 1)
	require.Error(t, err)

}

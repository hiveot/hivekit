package authenticators

import (
	"crypto/ed25519"
	"fmt"
	"log/slog"
	"time"

	"aidanwoods.dev/go-paseto"
	"github.com/hiveot/hivekit/go/modules/authn"
	"github.com/hiveot/hivekit/go/modules/authn/module/authnstore"
	"github.com/hiveot/hivekit/go/wot/td"
)

// PasetoAuthenticator for generating and validating session tokens.
// This implements the IAuthenticator interface
//
// Sessions are stored in-memory by their 'sessionStart' time.
type PasetoAuthenticator struct {
	// key used to create and verify session tokens
	signingKey ed25519.PrivateKey

	// authentication store for login verification
	clientStore authnstore.IAuthnStore

	// The URI of the authentication service that provides paseto tokens
	authServerURI             string
	AgentTokenValidityDays    int
	ConsumerTokenValidityDays int
	ServiceTokenValidityDays  int
}

// AddClient adds a client. This fails if the client already exists
// func (m *PasetoAuthenticator) AddClient(
// 	clientID string, displayName string, role string, pubKeyPem string) error {
// 	_, err := m.clientStore.GetProfile(clientID)
// 	if err == nil {
// 		return fmt.Errorf("Account for client '%s' already exists", clientID)
// 	}

// 	newProfile := authn.ClientProfile{
// 		ClientID:    clientID,
// 		DisplayName: displayName,
// 		Role:        role,
// 		PubKeyPem:   pubKeyPem,
// 	}
// 	return m.clientStore.Add(newProfile)
// }

// AddSecurityScheme adds this authenticator's security scheme to the given TD.
// This authenticator uses paseto tokens as bearer tokens that can be obtained from
// the login authentication service.
func (srv *PasetoAuthenticator) AddSecurityScheme(tdoc *td.TD) {

	// bearer security scheme for authenticating http and subprotocol connections
	format, alg := srv.GetAlg()

	tdoc.AddSecurityScheme("bearer_paseto", td.SecurityScheme{
		//AtType:        nil,
		Description: "Bearer token authentication",
		//Descriptions:  nil,
		//Proxy:         "",
		Scheme:        "bearer",          // nosec, basic, digest, bearer, psk, oauth2, apikey or auto
		Authorization: srv.authServerURI, // service to obtain a token
		Name:          "authorization",
		Alg:           alg,
		Format:        format,   // jwe, cwt, jws, jwt, paseto
		In:            "header", // query, body, cookie, uri, auto
	})
}

// CreateSessionToken creates a new token for the client
//
//	clientID is the account ID of a known client
//	validity is the token validity period.
//
// This returns the token
func (svc *PasetoAuthenticator) CreateToken(clientID string, validity time.Duration) (
	token string, validUntil time.Time, err error) {

	profile, err := svc.clientStore.GetProfile(clientID)
	if err != nil {
		return "", validUntil, err
	} else if validity == 0 {
		return "", validUntil, fmt.Errorf("CreateToken: validity cannot be 0")
	}

	// TODO: add support for nonce challenge with client pubkey

	// CreateToken creates a signed Paseto session token for a client.
	// The token is signed with the given signing key-pair and valid for the given duration.
	createdTime := time.Now()
	expiryTime := createdTime.Add(validity)

	pToken := paseto.NewToken()
	pToken.SetIssuer("hiveot")
	pToken.SetSubject(clientID)
	pToken.SetExpiration(expiryTime)
	pToken.SetIssuedAt(createdTime)
	pToken.SetNotBefore(createdTime)
	// custom claims
	pToken.SetString("clientID", clientID)
	pToken.SetString("role", string(profile.Role))

	secretKey, err := paseto.NewV4AsymmetricSecretKeyFromEd25519(svc.signingKey)
	if err != nil {
		slog.Error("failed making paseto secret key from ED25519")
		secretKey = paseto.NewV4AsymmetricSecretKey()
	}
	signedToken := pToken.V4Sign(secretKey, nil)

	expiration, _ := pToken.GetExpiration()
	validUntil = expiration.Local()
	return signedToken, validUntil, err

}

// DecodeSessionToken verifies the given token and returns its claims.
// optionally verify the signed nonce using the client's public key. (todo)
// This returns the auth info stored in the token.
//
// nonce based verification to prevent replay attacks is intended for future version.
//
// token is the token string containing a session token
// This returns the authenticated clientID stored in the token and its expiry time,
// or an error if invalid.
func (svc *PasetoAuthenticator) DecodeToken(
	sessionKey string, signedNonce string, nonce string) (
	clientID string, role string, issuedAt time.Time, validUntil time.Time, err error) {
	var pToken *paseto.Token

	pasetoParser := paseto.NewParserForValidNow()
	pubKey := svc.signingKey.Public().(ed25519.PublicKey)
	v4PubKey, err := paseto.NewV4AsymmetricPublicKeyFromEd25519(pubKey)
	if err == nil {
		pToken, err = pasetoParser.ParseV4Public(v4PubKey, sessionKey, nil)
	}
	if err == nil {
		clientID, err = pToken.GetString("clientID")
	}
	if err == nil {
		role, err = pToken.GetString("role")
	}
	if err == nil {
		issuedAt, err = pToken.GetIssuedAt()
		validUntil, err = pToken.GetExpiration()
	}
	if err != nil {
		slog.Warn("DecodeSessionToken: the given session token is no longer valid: ", "err", err.Error())
	}
	return clientID, role, issuedAt, validUntil, err
}

// GetAlg returns the authentication scheme and algorithm
func (svc *PasetoAuthenticator) GetAlg() (string, string) {
	return "paseto", "public"
}

// RefreshToken requests a new token based on the old token
// // This requires that the existing session is still valid
// func (svc *PasetoAuthenticator) RefreshToken(senderID string, oldToken string) (
// 	newToken string, validUntil time.Time, err error) {

// 	// validation only succeeds if there is an active session
// 	tokenClientID, _, _, _, err := svc.ValidateToken(oldToken)
// 	if err != nil || senderID != tokenClientID {
// 		return newToken, validUntil, fmt.Errorf("Invalid token or senderID mismatch")
// 	}
// 	// must still be a valid client
// 	prof, err := svc.clientStore.GetProfile(senderID)
// 	_ = prof
// 	if err != nil || prof.Disabled {
// 		return newToken, validUntil, fmt.Errorf("Profile for '%s' is disabled", senderID)
// 	}
// 	validityDays := svc.ConsumerTokenValidityDays
// 	if prof.Role == authn.ClientRoleAgent {
// 		validityDays = svc.AgentTokenValidityDays
// 	} else if prof.Role == authn.ClientRoleService {
// 		validityDays = svc.ServiceTokenValidityDays
// 	}
// 	validity := time.Duration(validityDays) * 24 * time.Hour
// 	newToken, validUntil, err = svc.CreateToken(senderID, validity)
// 	return newToken, validUntil, err
// }

// SetAuthServerURI this sets the server endpoint starting the authorization flow.
// This is included when adding the TD security scheme in AddSecurityScheme()
func (svc *PasetoAuthenticator) SetAuthServerURI(serverURI string) {
	svc.authServerURI = serverURI
}

// // update the client's password
// func (svc *PasetoAuthenticator) SetPassword(clientID, password string) error {
// 	return svc.clientStore.SetPassword(clientID, password)
// }

// func (svc *PasetoAuthenticator) ValidatePassword(clientID, password string) (err error) {
// 	clientProfile, err := svc.clientStore.VerifyPassword(clientID, password)
// 	_ = clientProfile
// 	return err
// }

// ValidateToken verifies the token and client are valid.
func (svc *PasetoAuthenticator) ValidateToken(token string) (
	clientID string, role string, issuedAt time.Time, validUntil time.Time, err error) {

	clientID, role, issuedAt, validUntil, err = svc.DecodeToken(token, "", "")

	if err != nil {
		return clientID, role, issuedAt, validUntil, err
	}
	// must still be a valid client
	prof, err := svc.clientStore.GetProfile(clientID)
	if err != nil || prof.Disabled {
		return clientID, role, issuedAt, validUntil, fmt.Errorf("Profile for '%s' is disabled", clientID)
	}

	return clientID, role, issuedAt, validUntil, nil
}

// NewPasetoAuthenticator returns a new instance of a Paseto token authenticator using the given signing key
// the session manager is used
func NewPasetoAuthenticator(
	authnStore authnstore.IAuthnStore,
	signingKey ed25519.PrivateKey) *PasetoAuthenticator {

	paseto.NewV4AsymmetricSecretKey()

	svc := &PasetoAuthenticator{
		signingKey:  signingKey,
		clientStore: authnStore,
		//authServerURI: authServerURI, use SetAuthServerURI
		// validity can be changed by user of this service
		AgentTokenValidityDays:    authn.DefaultAgentTokenValidityDays,
		ConsumerTokenValidityDays: authn.DefaultConsumerTokenValidityDays,
		ServiceTokenValidityDays:  authn.DefaultServiceTokenValidityDays,
	}
	var _ IAuthenticator = svc // interface check
	return svc
}

// NewPasetoAuthenticatorFromFile returns a new instance of a Paseto token authenticator
// loading a keypair from file or creating one if it doesn't exist.
// This returns nil if no signing key can be loaded or created
//
// The authServerURI is included the TD security scheme to point consumers to the
// endpoint to obtain tokens for this authenticator.
// func NewPasetoAuthenticatorFromFile(
// 	authnStore authnstore.IAuthnStore, keysDir string) *PasetoAuthenticator {

// 	clientID := "authn"
// 	authKey, err := keys.LoadCreateKeyPair(clientID, keysDir, keys.KeyTypeEd25519)

// 	if err != nil {
// 		slog.Error("NewPasetoAuthenticatorFromFile failed loading or creating a Paseto key pair",
// 			"err", err.Error(), "clientID", clientID)
// 		panic("failed loading or creating Paseto key pair")
// 	}
// 	signingKey := authKey.PrivateKey().(ed25519.PrivateKey)
// 	_ = err
// 	svc := NewPasetoAuthenticator(authnStore, signingKey)
// 	return svc
// }

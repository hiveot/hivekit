package authenticators

import (
	"crypto/ed25519"
	"fmt"
	"log/slog"
	"time"

	"aidanwoods.dev/go-paseto"
	"github.com/hiveot/hivekit/go/modules/authn"
	"github.com/hiveot/hivekit/go/modules/authn/service/authnstore"
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
	authnStore authnstore.IAuthnStore
	// The URI of the authentication service that provides paseto tokens
	authServerURI             string
	AgentTokenValidityDays    int
	ConsumerTokenValidityDays int
	ServiceTokenValidityDays  int

	// track session start, used in validation
	sessionStart map[string]time.Time

	// clients with their role, used in login
	clientStore authnstore.IAuthnStore
}

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

// CreateSessionToken creates a new session token for the client
//
//	clientID is the account ID of a known client
//	sessionID for which this token is valid. Use clientID to allow no session (agents)
//	validityS is the token validity period
//
// This returns the token
func (svc *PasetoAuthenticator) CreateToken(
	clientID string, role string, validity time.Duration) (token string, validUntil time.Time) {

	// TODO: add support for nonce challenge with client pubkey

	// CreateSessionToken creates a signed Paseto session token for a client.
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
	pToken.SetString("role", role)

	secretKey, err := paseto.NewV4AsymmetricSecretKeyFromEd25519(svc.signingKey)
	if err != nil {
		slog.Error("failed making paseto secret key from ED25519")
		secretKey = paseto.NewV4AsymmetricSecretKey()
	}
	signedToken := pToken.V4Sign(secretKey, nil)

	expiration, _ := pToken.GetExpiration()
	validUntil = expiration.Local()
	svc.sessionStart[clientID] = createdTime.Add(-time.Second)
	return signedToken, validUntil

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

// Login with password and generate a session token
// Intended for end-users that want to establish a session.
//
//	clientID is the client to log in
//	password to verify
//
// This returns a new token or an error if failed
func (svc *PasetoAuthenticator) Login(
	clientID string, password string) (token string, validUntil time.Time, err error) {

	// a user login always creates a session token
	err = svc.ValidatePassword(clientID, password)
	if err != nil {
		return "", validUntil, err
	}

	role, err := svc.authnStore.GetRole(clientID)
	if err != nil {
		return "", validUntil, err
	}

	// If a session start time does not exist yet, then record this as the session start.
	sessionStart, found := svc.sessionStart[clientID]
	if !found {
		sessionStart = time.Now()
		svc.sessionStart[clientID] = sessionStart
	}

	// create the session to allow token refresh
	validity := time.Hour * time.Duration(24*svc.ConsumerTokenValidityDays)
	token, validUntil = svc.CreateToken(clientID, role, validity)

	return token, validUntil, err
}

// Logout removes the client session
func (svc *PasetoAuthenticator) Logout(clientID string) {
	_, found := svc.sessionStart[clientID]
	if found {
		delete(svc.sessionStart, clientID)
	}
}

// RefreshToken requests a new token based on the old token
// This requires that the existing session is still valid
func (svc *PasetoAuthenticator) RefreshToken(
	senderID string, oldToken string) (newToken string, validUntil time.Time, err error) {

	// validation only succeeds if there is an active session
	tokenClientID, role, _, err := svc.ValidateToken(oldToken)
	if err != nil || senderID != tokenClientID {
		return newToken, validUntil, fmt.Errorf("Invalid token or senderID mismatch")
	}
	// must still be a valid client
	prof, err := svc.authnStore.GetProfile(senderID)
	_ = prof
	if err != nil || prof.Disabled {
		return newToken, validUntil, fmt.Errorf("Profile for '%s' is disabled", senderID)
	}
	validityDays := svc.ConsumerTokenValidityDays
	if prof.Role == authn.ClientRoleAgent {
		validityDays = svc.AgentTokenValidityDays
	} else if prof.Role == authn.ClientRoleService {
		validityDays = svc.ServiceTokenValidityDays
	}
	validity := time.Duration(validityDays) * 24 * time.Hour
	newToken, validUntil = svc.CreateToken(senderID, role, validity)
	return newToken, validUntil, nil
}

// SetAuthServerURI this sets the server endpoint starting the authorization flow.
// This is included when adding the TD security scheme in AddSecurityScheme()
func (svc *PasetoAuthenticator) SetAuthServerURI(serverURI string) {
	svc.authServerURI = serverURI
}

// update the client's password
func (svc *PasetoAuthenticator) SetPassword(clientID, password string) error {
	return svc.authnStore.SetPassword(clientID, password)
}

func (svc *PasetoAuthenticator) ValidatePassword(clientID, password string) (err error) {
	clientProfile, err := svc.authnStore.VerifyPassword(clientID, password)
	_ = clientProfile
	return err
}

// ValidateToken verifies the token and client are valid.
func (svc *PasetoAuthenticator) ValidateToken(token string) (
	clientID string, role string, validUntil time.Time, err error) {

	clientID, role, issuedAt, validUntil, err := svc.DecodeToken(token, "", "")

	if err != nil {
		return clientID, role, validUntil, err
	}
	// must still be a valid client
	prof, err := svc.authnStore.GetProfile(clientID)
	if err != nil || prof.Disabled {
		return clientID, role, validUntil, fmt.Errorf("Profile for '%s' is disabled", clientID)
	}
	// check the token is of an active client
	// this is set during CreateToken and Login
	sessionStart, found := svc.sessionStart[clientID]
	if !found {
		slog.Warn("ValidateToken. No valid session found for client", "clientID", clientID)
		return clientID, role, validUntil, fmt.Errorf("Session is no longer valid")
	}
	// the session must have started before the token was issued
	if issuedAt.Before(sessionStart) {
		slog.Warn("ValidateToken. The token session is no longer valid", "clientID", clientID)
		return clientID, role, validUntil, fmt.Errorf("Session is no longer valid")
	}

	return clientID, role, validUntil, nil
}

// NewPasetoAuthenticator returns a new instance of a Paseto token authenticator using the given signing key
// the session manager is used
func NewPasetoAuthenticator(
	authnStore authnstore.IAuthnStore,
	signingKey ed25519.PrivateKey) *PasetoAuthenticator {

	paseto.NewV4AsymmetricSecretKey()

	svc := PasetoAuthenticator{
		signingKey: signingKey,
		authnStore: authnStore,
		//authServerURI: authServerURI, use SetAuthServerURI
		// validity can be changed by user of this service
		AgentTokenValidityDays:    authn.DefaultAgentTokenValidityDays,
		ConsumerTokenValidityDays: authn.DefaultConsumerTokenValidityDays,
		ServiceTokenValidityDays:  authn.DefaultServiceTokenValidityDays,
		sessionStart:              make(map[string]time.Time),
	}
	return &svc
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

package authenticators

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/hiveot/hivekit/go/modules/authn"
	"github.com/hiveot/hivekit/go/modules/authn/module/authnstore"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/wot/td"
)

// JWTAuthenticator for generating and validating session tokens.
// This implements the IAuthenticator interface
//
// Sessions are stored in-memory by their 'sessionStart' time.
type JWTAuthenticator struct {
	// key used to create and verify session tokens
	signingKey *ecdsa.PrivateKey
	// authentication store for account verification
	authnStore authnstore.IAuthnStore
	//
	authServerURI string
	//
	AgentTokenValidityDays    int
	ConsumerTokenValidityDays int
	ServiceTokenValidityDays  int

	// track session start, used in validation
	sessionStart map[string]time.Time

	// signing method used
	signingMethod jwt.SigningMethod // default SigningMethodES256
}

// AddSecurityScheme adds the security scheme that this authenticator supports.
// http supports bearer tokens for request authentication, basic and digest authentication
// for logging in.
func (srv *JWTAuthenticator) AddSecurityScheme(tdoc *td.TD) {

	// bearer security scheme for authenticating http and subprotocol connections
	format, alg := srv.GetAlg()

	tdoc.AddSecurityScheme("bearer_jwt", td.SecurityScheme{
		//AtType:        nil,
		Description: "Bearer token authentication",
		//Descriptions:  nil,
		//Proxy:         "",
		Scheme:        "bearer",          // nosec, basic, digest, bearer, psk, oauth2, apikey or auto
		Authorization: srv.authServerURI, // authentication service URI
		Name:          "authorization",
		Alg:           alg,
		Format:        format,   // jwe, cwt, jws, jwt, paseto
		In:            "header", // query, body, cookie, uri, auto
	})
	// bearer security scheme for authenticating http digest connections
	// tbd. clients should login and use bearer tokens.
	//tdoc.AddSecurityScheme("digest_sc", td.SecurityScheme{
	//	Description: "Digest authentication",
	//	Scheme:      "digest", // nosec, basic, digest, bearer, psk, oauth2, apikey or auto
	//	In:          "body",   // query, header, body, cookie, uri, auto
	//})
}

// CreateSessionToken creates a new session token for the client
//
//	clientID is the account ID of a known client
//	sessionID for which this token is valid. Use clientID to allow no session (agents)
//	validity is the token validity period
//
// This returns the token
func (svc *JWTAuthenticator) CreateToken(
	clientID string, role string, validity time.Duration) (token string, validUntil time.Time) {

	// TODO: add support for nonce challenge with client pubkey

	// CreateSessionToken creates a signed JWT session token for a client.
	// The token is constructed with MapClaims containing "clientID" identifying
	// the authenticated client.
	// The token is signed with the given signing key-pair and valid for the given duration.
	createdTime := time.Now()
	expiryTime := createdTime.Add(validity)
	signingKeyPub, _ := x509.MarshalPKIXPublicKey(&svc.signingKey.PublicKey)
	signingKeyPubStr := base64.StdEncoding.EncodeToString(signingKeyPub)

	// Create the JWT claims, which includes the username, clientType and expiry time
	claims := jwt.MapClaims{
		"alg": jwt.SigningMethodES256,
		"typ": "JWT",
		//"aud": authInfo.SenderID, // recipient of the jwt
		"sub": clientID,          // subject of the jwt, eg the client ID
		"iss": signingKeyPubStr,  // issuer of the jwt (public key)
		"exp": expiryTime.Unix(), // expiry time. Seconds since epoch
		"iat": time.Now().Unix(), // issued at. Seconds since epoch
		// custom claims
		"role": role,
	}

	// Declare the token with the algorithm used for signing, and the claims
	claimsToken := jwt.NewWithClaims(svc.signingMethod, claims)
	sessionToken, _ := claimsToken.SignedString(svc.signingKey)
	svc.sessionStart[clientID] = createdTime.Add(-time.Second)
	return sessionToken, expiryTime
}

// DecodeSessionToken verifies the given JWT token and returns its claims.
// optionally verify the signed nonce using the client's public key.
// This returns the auth info stored in the token.
//
// nonce based verification to prevent replay attacks is intended for future version.
//
// token is the jwt token string containing a session token
// This returns the authenticated clientID stored in the token and its expiry time,
// or an error if invalid.
func (svc *JWTAuthenticator) DecodeToken(token string, signedNonce string, nonce string) (
	clientID string, role string, issuedAt time.Time, validUntil time.Time, err error) {

	signingKeyPub, _ := x509.MarshalPKIXPublicKey(&svc.signingKey.PublicKey)
	signingKeyPubStr := base64.StdEncoding.EncodeToString(signingKeyPub)

	claims := jwt.MapClaims{}
	jwtToken, err := jwt.ParseWithClaims(token, &claims,
		func(token *jwt.Token) (interface{}, error) {
			return &svc.signingKey.PublicKey, nil
		}, jwt.WithValidMethods([]string{
			jwt.SigningMethodES256.Name,
			jwt.SigningMethodES384.Name,
			jwt.SigningMethodES512.Name,
			"EdDSA",
		}),
		jwt.WithIssuer(signingKeyPubStr), // url encoded string
		jwt.WithExpirationRequired(),
	)
	if err == nil {
		clientID, err = claims.GetSubject()
	}
	if err == nil {
		roleClaim := claims["role"]
		if roleClaim != nil {
			role = roleClaim.(string)
		}
	}
	if err == nil {
		var expiryTime *jwt.NumericDate
		expiryTime, err = claims.GetExpirationTime()
		if expiryTime != nil {
			validUntil = expiryTime.Time
		}
	}
	if err == nil {
		var issuedTime *jwt.NumericDate
		issuedTime, err = claims.GetIssuedAt()
		if issuedTime != nil {
			issuedAt = issuedTime.Time
		}
	}
	if err == nil && (!jwtToken.Valid || role == "" || clientID == "") {
		err = fmt.Errorf("Invalid token or missing role or clientID")
	}
	return clientID, role, issuedAt, validUntil, err
}

// GetAlg returns the authentication scheme (jwt) and algorithm
func (svc *JWTAuthenticator) GetAlg() (string, string) {
	return "jwt", svc.signingMethod.Alg()
}

// Login with password and generate a session token
// Intended for end-users that want to establish a session.
//
//	clientID is the client to log in
//	password to verify
//
// This returns a session token, its session ID, or an error if failed
func (svc *JWTAuthenticator) Login(
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
func (svc *JWTAuthenticator) Logout(clientID string) {
	_, found := svc.sessionStart[clientID]
	if found {
		delete(svc.sessionStart, clientID)
	}
}

// RefreshToken requests a new token based on the old token
// This requires that the existing session is still valid
func (svc *JWTAuthenticator) RefreshToken(
	senderID string, oldToken string) (newToken string, validUntil time.Time, err error) {

	// validation only succeeds if there is an active session
	tokenClientID, role, _, err := svc.ValidateToken(oldToken)
	if err != nil || senderID != tokenClientID {
		return newToken, validUntil, fmt.Errorf("Invalid token or senderID mismatch")
	}
	// must still be a valid client
	prof, err := svc.authnStore.GetProfile(senderID)
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

// SetAuthServerURI this sets the server endpoint needed to login.
// This is included when adding the TD security scheme in AddSecurityScheme()
func (svc *JWTAuthenticator) SetAuthServerURI(serverURI string) {
	svc.authServerURI = serverURI
}
func (svc *JWTAuthenticator) ValidatePassword(clientID, password string) (err error) {
	clientProfile, err := svc.authnStore.VerifyPassword(clientID, password)
	_ = clientProfile
	return err
}

// update the client's password
func (svc *JWTAuthenticator) SetPassword(clientID, password string) error {
	return svc.authnStore.SetPassword(clientID, password)
}

// ValidateToken verifies the token and client are valid.
func (svc *JWTAuthenticator) ValidateToken(token string) (
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
	if issuedAt.Before(sessionStart) {
		slog.Warn("ValidateToken. The token session is no longer valid", "clientID", clientID)
		return clientID, role, validUntil, fmt.Errorf("Session is no longer valid")
	}

	return clientID, role, validUntil, nil
}

// NewJWTAuthenticator returns a new instance of a JWT token authenticator
func NewJWTAuthenticator(
	authnStore authnstore.IAuthnStore, signingKey *ecdsa.PrivateKey, authServerURI string) *JWTAuthenticator {
	svc := &JWTAuthenticator{
		signingKey:    signingKey,
		authnStore:    authnStore,
		authServerURI: authServerURI,
		// validity can be changed by user of this service
		AgentTokenValidityDays:    authn.DefaultAgentTokenValidityDays,
		ConsumerTokenValidityDays: authn.DefaultConsumerTokenValidityDays,
		ServiceTokenValidityDays:  authn.DefaultServiceTokenValidityDays,
		signingMethod:             jwt.SigningMethodES256,
		sessionStart:              make(map[string]time.Time),
	}
	var _ transports.IAuthenticator = svc // interface check
	return svc
}

// NewJWTAuthenticatorFromFile returns a new instance of a JWT token authenticator
// loading a keypair from file or creating one if it doesn't exist.
// This returns nil if no signing key can be loaded or created
//func NewJWTAuthenticatorFromFile(
//	authnStore api.IAuthnStore,
//	keysDir string, keyType keys.KeyType) *JWTAuthenticator {
//
//	clientID := "authn"
//	signingKey, err := keys.LoadCreateKeyPair(clientID, keysDir, keyType)
//	if err != nil {
//		slog.Error("NewJWTAuthenticatorFromFile failed creating key pair for client",
//			"err", err.Error(), "clientID", clientID)
//		return nil
//	}
//	_ = err
//	svc := NewJWTAuthenticator(authnStore, signingKey)
//	return svc
//}

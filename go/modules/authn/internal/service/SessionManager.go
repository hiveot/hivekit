package service

import (
	"crypto/ed25519"
	"fmt"
	"log/slog"
	"time"

	"github.com/hiveot/hivekit/go/api/td"
	authnapi "github.com/hiveot/hivekit/go/modules/authn/api"
	"github.com/hiveot/hivekit/go/modules/authn/internal/authenticators"
	authnstore "github.com/hiveot/hivekit/go/modules/authn/internal/store"
	"github.com/hiveot/hivekit/go/utils"
)

// Session manager for authenticating users.
// This implements the IAuthenticator and IAuthnAuthenticator interfaces
type SessionManager struct {
	// Auth token validity for agents in days
	AgentTokenValidityDays int `yaml:"agentTokenValidityDays,omitempty"`
	// Auth token validity for consumers in days
	ConsumerTokenValidityDays int `yaml:"consumerTokenValidityDays,omitempty"`
	// Auth token validity for services in days
	ServiceTokenValidityDays int `yaml:"serviceTokenValidityDays,omitempty"`

	//
	authnStore authnstore.IAuthnStore

	// The primary authenticator
	authenticator authenticators.IAuthnAuthenticator

	// track session start, used in validation
	sessionStart map[string]time.Time
}

// AddSecurityScheme adds the authenticator's security scheme to the given TD.
func (sm *SessionManager) AddSecurityScheme(tdoc *td.TD) {
	sm.authenticator.AddSecurityScheme(tdoc)
}

// Return the authenticator
// func (m *AuthnModule) GetAuthenticator() authenticators.IAuthenticator {
// 	return m.authenticator
// }

// CreateToken creates a new session token for the client using the configured authenticator.
//
// This creates a session that is valid until logout.
//
//	clientID is the account ID of a known client
//	validity is the token validity period.
//
// This returns the token
func (sm *SessionManager) CreateToken(clientID string, validity time.Duration) (
	token string, validUntil time.Time, err error) {

	//
	createdTime := time.Now()
	sm.sessionStart[clientID] = createdTime.Add(-time.Second)

	token, validUntil, err = sm.authenticator.CreateToken(clientID, validity)
	return
}

// DecodeToken decodes the given token using the configured authenticator.
// optionally verify the signed nonce using the client's public key.
// This returns the auth info stored in the token.
func (sm *SessionManager) DecodeToken(token string, signedNonce string, nonce string) (
	clientID string, issuedAt time.Time, validUntil time.Time, err error) {
	return sm.authenticator.DecodeToken(token, signedNonce, nonce)
}

// Login with password and generate a session token
// Intended for end-users that want to establish a session.
//
//	clientID is the client to log in
//	password to verify
//
// This returns a session token, its session ID, or an error if failed
func (sm *SessionManager) Login(
	clientID string, password string) (token string, validUntil time.Time, err error) {

	// a user login always creates a session token
	err = sm.ValidatePassword(clientID, password)
	if err != nil {
		return "", validUntil, err
	}

	// If a session start time does not exist yet, then record this as the session start.
	sessionStart, found := sm.sessionStart[clientID]
	if !found {
		sessionStart = time.Now().Add(-time.Second) // prevent comparison with token iat failing
		sm.sessionStart[clientID] = sessionStart
	}

	// create the session to allow token refresh
	validity := time.Hour * time.Duration(24*sm.ConsumerTokenValidityDays)
	token, validUntil, _ = sm.authenticator.CreateToken(clientID, validity)

	return token, validUntil, err
}

// Logout removes the client session
func (sm *SessionManager) Logout(clientID string) {
	_, found := sm.sessionStart[clientID]
	if found {
		delete(sm.sessionStart, clientID)
	}
}

// RefreshToken requests a new token based on the old token
// This requires that the existing session is still valid
func (sm *SessionManager) RefreshToken(senderID string, oldToken string) (
	newToken string, validUntil time.Time, err error) {

	// validation only succeeds if there is an active session
	tokenClientID, _, _, err := sm.ValidateToken(oldToken)
	if err != nil || senderID != tokenClientID {
		return newToken, validUntil, fmt.Errorf("Invalid token or senderID mismatch")
	}
	// must still be a valid client
	prof, err := sm.authnStore.GetProfile(senderID)
	_ = prof
	if err != nil || prof.Disabled {
		return newToken, validUntil, fmt.Errorf("Profile for '%s' is disabled", senderID)
	}
	validityDays := sm.ConsumerTokenValidityDays
	if prof.Role == authnapi.ClientRoleAgent {
		validityDays = sm.AgentTokenValidityDays
	} else if prof.Role == authnapi.ClientRoleService {
		validityDays = sm.ServiceTokenValidityDays
	}
	validity := time.Duration(validityDays) * 24 * time.Hour
	newToken, validUntil, err = sm.authenticator.CreateToken(senderID, validity)
	return newToken, validUntil, err
}

// validate if the password is valid to login with
func (sm *SessionManager) ValidatePassword(clientID, password string) (err error) {
	clientProfile, err := sm.authnStore.VerifyPassword(clientID, password)
	_ = clientProfile
	return err
}

// ValidateToken verifies the token and client are valid.
func (sm *SessionManager) ValidateToken(token string) (
	clientID string, issuedAt time.Time, validUntil time.Time, err error) {

	clientID, issuedAt, validUntil, err = sm.authenticator.ValidateToken(token)
	if err != nil {
		return
	}

	// check the token is of an active client
	// this is set during CreateToken and Login
	sessionStart, found := sm.sessionStart[clientID]
	if !found {
		slog.Warn("ValidateToken. No valid session found for client", "clientID", clientID)
		return clientID, issuedAt, validUntil, fmt.Errorf("Session is no longer valid")
	}
	// the session must have started before the token was issued
	// this allows a session restart to invalidate all old tokens
	if issuedAt.Before(sessionStart) {
		slog.Warn("ValidateToken. The token session is no longer valid", "clientID", clientID)
		return clientID, issuedAt, validUntil, fmt.Errorf("Session is no longer valid")
	}

	return clientID, issuedAt, validUntil, err
}

// Start a new session manager for client sessions
func StartSessionManager(
	authnStore authnstore.IAuthnStore, keysDir string) (*SessionManager, error) {

	clientID := "authn"

	// store the signing key
	signingPrivKey, _, err := utils.LoadCreateKeyPair(
		clientID, keysDir, utils.KeyTypeED25519)
	if err != nil {
		return nil, err
	}

	sm := &SessionManager{
		authnStore:                authnStore,
		AgentTokenValidityDays:    authnapi.DefaultAgentTokenValidityDays,
		ServiceTokenValidityDays:  authnapi.DefaultServiceTokenValidityDays,
		ConsumerTokenValidityDays: authnapi.DefaultConsumerTokenValidityDays,
		sessionStart:              make(map[string]time.Time),
	}

	sm.authenticator = authenticators.NewPasetoAuthenticator(
		sm.authnStore, signingPrivKey.(ed25519.PrivateKey))

	var _ authnapi.ISessionManager = sm // interface check
	return sm, nil
}

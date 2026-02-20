package tptests

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hiveot/hivekit/go/wot/td"
	"github.com/teris-io/shortid"
)

// DummyAuthenticator for testing the transport protocol bindings
// This implements the IAuthenticator interface.
type DummyAuthenticator struct {
	passwords map[string]string
	// flag whether sessions are valid for this client
	inSession     map[string]string
	authServerURI string
}

// AddClient adds a test client and return an auth token
func (d *DummyAuthenticator) AddClient(clientID string, role string, password string, pubKey string) error {
	d.passwords[clientID] = password

	token, validUntil, err := d.CreateToken(clientID, 0)
	_ = validUntil
	d.inSession[clientID] = token
	return err
}

// AddSecurityScheme adds the security scheme that this authenticator supports.
func (srv *DummyAuthenticator) AddSecurityScheme(tdoc *td.TD) {

	// bearer security scheme for authenticating http and subprotocol connections
	format, alg := srv.GetAlg()

	tdoc.AddSecurityScheme("bearer", td.SecurityScheme{
		//AtType:        nil,
		Description: "JWT dummy token authentication",
		//Descriptions:  nil,
		//Proxy:         "",
		Scheme:        "bearer", // nosec, basic, digest, bearer, psk, oauth2, apikey or auto
		Authorization: srv.authServerURI,
		Name:          "authorization",
		Alg:           alg,
		Format:        format,   // jwe, cwt, jws, jwt, paseto
		In:            "header", // query, body, cookie, uri, auto
	})
}

//func (d *DummyAuthenticator) AddToken(clientID string, token string) {
//	d.tokens[clientID] = token
//}

// if validity is 0 it defaults to 1 minute
func (d *DummyAuthenticator) CreateToken(
	clientID string, validity time.Duration) (token string, validUntil time.Time, err error) {
	if validity == 0 {
		validity = time.Minute
	}

	_, isClient := d.passwords[clientID]
	if !isClient {
		return "", validUntil, fmt.Errorf("Unknown client %s", clientID)
	}

	authkeytoken := shortid.MustGenerate()
	validUntil = time.Now().Add(validity)
	// the real auth knows the role from adding clients
	token = fmt.Sprintf("%s/%s/%s", clientID, "role", authkeytoken)
	// simulate a session with the tokens map
	d.inSession[clientID] = token
	return token, validUntil, nil
}

func (d *DummyAuthenticator) DecodeToken(token string, signedNonce string, nonce string) (
	clientID string, issuedAt, validUntil time.Time, err error) {

	// fake it
	issuedAt = time.Now().Add(-time.Minute)
	// validUntil = time.Now().Add(time.Minute)
	clientID, validUntil, err = d.ValidateToken(token)

	return clientID, issuedAt, validUntil, err
}

// GetAlg pretend to use jwt
func (d *DummyAuthenticator) GetAlg() (string, string) {
	return "jwt", "es256"
}

func (d *DummyAuthenticator) Login(
	clientID string, password string) (token string, validUntil time.Time, err error) {
	currPass, isClient := d.passwords[clientID]
	if isClient && currPass == password {
		token, validUntil, _ = d.CreateToken(clientID, 0)
		d.inSession[clientID] = token
		return token, validUntil, nil
	}
	return "", validUntil, fmt.Errorf("invalid login")
}

func (d *DummyAuthenticator) Logout(clientID string) {
	delete(d.inSession, clientID)
}

func (d *DummyAuthenticator) ValidatePassword(clientID string, password string) (err error) {
	currPass, isClient := d.passwords[clientID]
	if isClient && currPass == password {
		return nil
	}
	return errors.New("bad login or pass")
}

func (d *DummyAuthenticator) RefreshToken(
	senderID string, oldToken string) (newToken string, validUntil time.Time, err error) {

	tokenClientID, validUntil, err := d.ValidateToken(oldToken)
	if err != nil || senderID != tokenClientID {
		err = fmt.Errorf("invalid token, client or sender")
	} else {
		newToken, validUntil, _ = d.CreateToken(senderID, 0)
	}
	return newToken, validUntil, err
}

func (d *DummyAuthenticator) SetAuthServerURI(authServerURI string) {
	d.authServerURI = authServerURI
}

// update the client's password
func (d *DummyAuthenticator) SetPassword(clientID, password string) error {
	return nil
}

// Validate the token
func (d *DummyAuthenticator) ValidateToken(token string) (
	clientID string, validUntil time.Time, err error) {

	parts := strings.Split(token, "/")
	if len(parts) != 3 {
		return "", validUntil, fmt.Errorf("badToken")
	}
	clientID = parts[0]

	// simulate a session by checking if a recent token was issued
	_, found := d.inSession[clientID]
	if !found {
		err = errors.New("no active session")
	}

	return clientID, validUntil, err
}

func NewDummyAuthenticator() *DummyAuthenticator {
	d := &DummyAuthenticator{
		passwords: make(map[string]string),
		inSession: make(map[string]string),
	}
	// var _ transports.IAuthValidator = d // interface check
	return d
}

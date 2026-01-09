package authnapi

import (
	"crypto/x509"
	"log/slog"
	"net/http"

	"github.com/hiveot/hivekit/go/lib/servers/httpbasic"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/modules/transports/authn"
	"github.com/hiveot/hivekit/go/modules/transports/httptransport"
	"github.com/hiveot/hivekit/go/modules/transports/httptransport/httpapi"
	jsoniter "github.com/json-iterator/go"
)

// AuthnClient is a client for authentication operations such as login
// Use of the AuthnClient object is optional - the functions can be used directly.
type AuthnClient struct {
	tlsClient httptransport.ITlsClient
	clientID  string
}

func NewAuthnClient(clientID string, serverURL string, caCert *x509.Certificate) *AuthnClient {
	tlsClient := httpapi.NewTLSClient(serverURL, nil, caCert, 0)
	return &AuthnClient{
		tlsClient: tlsClient,
		clientID:  clientID,
	}
}

func (cl *AuthnClient) LoginWithPassword(password string) (newToken string, err error) {
	newToken, err = LoginWithPassword(cl.tlsClient, cl.clientID, password)
	return newToken, err
}

func (cl *AuthnClient) Logout(token string) (err error) {
	err = Logout(cl.tlsClient, token)
	return err
}

func (cl *AuthnClient) RefreshToken(oldToken string) (newToken string, err error) {
	newToken, err = RefreshToken(cl.tlsClient, cl.clientID, oldToken)
	return newToken, err
}

func LoginWithPassword(tlsClient httptransport.ITlsClient, clientID, password string) (newToken string, err error) {

	// LoginWithPassword posts a login request to the TLS server using a login ID and
	// password and obtain an auth token for use with ConnectWithToken.
	// This uses the http-basic login endpoint.
	//
	// FIXME: use a WoT standardized auth method
	//
	slog.Info("ConnectWithPassword", "clientID", clientID)

	// FIXME: figure out how a standard login method is used to obtain an auth token
	args := transports.UserLoginArgs{
		Login:    clientID,
		Password: password,
	}

	argsJSON, _ := jsoniter.Marshal(args)
	loginPath := authn.HttpPostLoginPath
	outputRaw, status, err := tlsClient.Post(loginPath, []byte(argsJSON))

	if err == nil && status == http.StatusOK {
		err = jsoniter.Unmarshal(outputRaw, &newToken)
	}
	if err != nil {
		slog.Warn("AuthenticateWithPassword failed: " + err.Error())
	}
	return newToken, err
}

// LoginWithForm invokes login using a form - temporary helper
// intended for testing a connection to a web server.
//
// This sets the bearer token for further requests. It requires the server
// to set a session cookie in response to the login.
//func (cl *HiveotSseClient) LoginWithForm(
//	password string) (newToken string, err error) {
//
//	// FIXME: does this client need a cookie jar???
//	formMock := url.Values{}
//	formMock.Add("loginID", cl.GetClientID())
//	formMock.Add("password", password)
//
//	var loginHRef string
//	f := cl.getForm(wot.HTOpLoginWithForm, "", "")
//	if f != nil {
//		loginHRef, _ = f.GetHRef()
//	}
//	loginURL, err := url.Parse(loginHRef)
//	if err != nil {
//		return "", err
//	}
//	if loginURL.Host == "" {
//		loginHRef = cl.fullURL + loginHRef
//	}
//
//	//PostForm should return a cookie that should be used in the http connection
//	if loginHRef == "" {
//		return "", errors.New("Login path not found in getForm")
//	}
//	resp, err := cl.httpClient.PostForm(loginHRef, formMock)
//	if err != nil {
//		return "", err
//	}
//
//	// get the session token from the cookie
//	//cookie := resp.Request.Header.Get("cookie")
//	cookie := resp.Header.Get("cookie")
//	kvList := strings.Split(cookie, ",")
//
//	for _, kv := range kvList {
//		kvParts := strings.SplitN(kv, "=", 2)
//		if kvParts[0] == "session" {
//			cl.bearerToken = kvParts[1]
//			break
//		}
//	}
//	if cl.bearerToken == "" {
//		slog.Error("No session cookie was received on login")
//	}
//	return cl.bearerToken, err
//}

// Logout requests invalidates the token and closes the connection
// tlsClient is a client with an existing authenticated connection
func Logout(tlsClient httptransport.ITlsClient, token string) (err error) {

	logoutPath := authn.HttpPostLogoutPath
	_, _, err = tlsClient.Post(logoutPath, nil)
	tlsClient.Close()
	return err
}

// RefreshToken invokes the hub's authenticator to refresh the token.
//
// tlsClient is a client with an existing authenticated connection
//
// This returns a new authentication token, or an error
func RefreshToken(tlsClient httptransport.ITlsClient, clientID string, oldToken string) (newToken string, err error) {
	refreshPath := httpbasic.HttpPostRefreshPath
	dataJSON, _ := jsoniter.Marshal(oldToken)
	tlsClient.ConnectWithToken(clientID, oldToken)
	outputRaw, status, err := tlsClient.Post(refreshPath, dataJSON)

	if err == nil && status == http.StatusOK {
		err = jsoniter.Unmarshal(outputRaw, &newToken)
	}

	if err != nil {
		slog.Warn("RefreshToken failed: " + err.Error())
	} else {
		// apply the token to the connection
		err = tlsClient.ConnectWithToken(clientID, oldToken)
	}
	return newToken, err
}

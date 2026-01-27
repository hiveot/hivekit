package authnclient

import (
	"crypto/x509"
	"log/slog"
	"net/http"
	"net/url"

	authnservice "github.com/hiveot/hivekit/go/modules/authn/service"
	"github.com/hiveot/hivekit/go/modules/transports"
	tlsclient "github.com/hiveot/hivekit/go/modules/transports/httpserver/client"
	jsoniter "github.com/json-iterator/go"
)

// AuthnUserHttpClient is a client for authentication operations such as login using http requests.
type AuthnUserHttpClient struct {
	tlsClient transports.ITlsClient
}

// Close the underlying TLS client used by the authentication client
func (cl *AuthnUserHttpClient) Close() {
	cl.tlsClient.Close()
}

// set the clientID and authn token this client uses
func (cl *AuthnUserHttpClient) ConnectWithToken(clientID string, token string) (err error) {

	cl.tlsClient.ConnectWithToken(clientID, token)
	return nil
}

// Return the TLS client used to connect to the authn server.
// This can be used anywhere an http client is needed for the same server.
func (cl *AuthnUserHttpClient) GetTlsClient() transports.ITlsClient {
	return cl.tlsClient
}

func (cl *AuthnUserHttpClient) LoginWithPassword(clientID string, password string) (newToken string, err error) {

	newToken, err = LoginWithPassword(
		cl.tlsClient, clientID, password)
	return newToken, err
}

func (cl *AuthnUserHttpClient) Logout(token string) (err error) {
	err = Logout(cl.tlsClient, token)
	return err
}

func (cl *AuthnUserHttpClient) RefreshToken(oldToken string) (newToken string, err error) {
	newToken, err = RefreshToken(
		cl.tlsClient, cl.tlsClient.GetClientID(), oldToken)
	return newToken, err
}

func NewAuthnHttpClient(serverURL string, caCert *x509.Certificate) *AuthnUserHttpClient {
	parts, err := url.Parse(serverURL)
	if err != nil {
		slog.Error("NewAuthnClient: invalid server URL", "err", err.Error())
		return nil
	}

	tlsClient := tlsclient.NewTLSClient(parts.Host, nil, caCert, 0)
	return &AuthnUserHttpClient{
		tlsClient: tlsClient,
	}
}

func LoginWithPassword(tlsClient transports.ITlsClient, clientID, password string) (newToken string, err error) {

	// LoginWithPassword posts a login request to the TLS server using a login ID and
	// password and obtain an auth token for use with ConnectWithToken.
	// This uses the http-basic login endpoint.
	//
	// FIXME: use a WoT standardized auth method
	//
	slog.Info("LoginWithPassword", "clientID", clientID)

	// FIXME: figure out how a standard login method is used to obtain an auth token
	args := authnservice.UserLoginArgs{
		UserName: clientID,
		Password: password,
	}

	argsJSON, _ := jsoniter.Marshal(args)
	loginPath := authnservice.HttpPostLoginPath
	outputRaw, status, err := tlsClient.Post(loginPath, []byte(argsJSON))

	if err == nil && status == http.StatusOK {
		err = jsoniter.Unmarshal(outputRaw, &newToken)
		// apply the new token in this client
		tlsClient.ConnectWithToken(clientID, newToken)
	}
	if err != nil {
		slog.Warn("LoginWithPassword failed: " + err.Error())
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
func Logout(tlsClient transports.ITlsClient, token string) (err error) {

	logoutPath := authnservice.HttpPostLogoutPath
	_, _, err = tlsClient.Post(logoutPath, nil)
	tlsClient.Close()
	return err
}

// RefreshToken invokes the hub's authenticator to refresh the token.
//
// tlsClient is a client with an existing authenticated connection
//
// This returns a new authentication token, or an error
func RefreshToken(tlsClient transports.ITlsClient, clientID string, oldToken string) (newToken string, err error) {
	refreshPath := authnservice.HttpPostRefreshPath
	dataJSON, _ := jsoniter.Marshal(oldToken)
	// first initialize the client with the old token
	tlsClient.ConnectWithToken(clientID, oldToken)
	// post a request and expect an instance response
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

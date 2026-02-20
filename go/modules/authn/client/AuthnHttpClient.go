package authnclient

import (
	"crypto/x509"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/hiveot/hivekit/go/modules/authn"
	"github.com/hiveot/hivekit/go/modules/authn/server"
	"github.com/hiveot/hivekit/go/modules/transports"
	tlsclient "github.com/hiveot/hivekit/go/modules/transports/httpserver/client"
	jsoniter "github.com/json-iterator/go"
)

// AuthnHttpClient is a client for authentication operations such as login using http requests.
// This is a simple API for clients to be able to obtain an auth token and refresh it.
type AuthnHttpClient struct {
	tlsClient transports.ITlsClient
}

// Close the underlying TLS client used by the authentication client
func (cl *AuthnHttpClient) Close() {
	cl.tlsClient.Close()
}

// set the clientID and authn token this client uses
func (cl *AuthnHttpClient) ConnectWithToken(clientID string, token string) (err error) {

	cl.tlsClient.ConnectWithToken(clientID, token)
	return nil
}

// Return the client's profile.
// The client must be authenticated first.
func (cl *AuthnHttpClient) GetProfile() (profile authn.ClientProfile, err error) {
	getProfilePath := server.HttpGetProfilePath
	outputRaw, status, err := cl.tlsClient.Get(getProfilePath)

	if err == nil && status == http.StatusOK {
		err = jsoniter.Unmarshal(outputRaw, &profile)
	}
	return profile, err
}

// Return the TLS client used to connect to the authn server.
// This can be used anywhere an http client is needed for the same server.
func (cl *AuthnHttpClient) GetTlsClient() transports.ITlsClient {
	return cl.tlsClient
}

func (cl *AuthnHttpClient) LoginWithPassword(clientID string, password string) (newToken string, err error) {

	// LoginWithPassword posts a login request to the TLS server using a login ID and
	// password and obtain an auth token for use with ConnectWithToken.
	// This uses the http-basic login endpoint.
	//
	// TBD: is there a WoT standardized auth method?
	//
	slog.Info("LoginWithPassword", "clientID", clientID)

	args := server.UserLoginArgs{
		UserName: clientID,
		Password: password,
	}

	argsJSON, _ := jsoniter.Marshal(args)
	loginPath := server.HttpPostLoginPath
	outputRaw, status, err := cl.tlsClient.Post(loginPath, []byte(argsJSON))

	if err == nil && status == http.StatusOK {
		err = jsoniter.Unmarshal(outputRaw, &newToken)
		// apply the new token in this client
		cl.tlsClient.ConnectWithToken(clientID, newToken)
	}
	if err != nil {
		slog.Warn("LoginWithPassword failed: " + err.Error())
	}
	return newToken, err

}

func (cl *AuthnHttpClient) Logout(token string) (err error) {

	logoutPath := server.HttpPostLogoutPath
	_, _, err = cl.tlsClient.Post(logoutPath, nil)
	cl.tlsClient.Close()
	return err
}

// Use the http address to request a token refresh
func (cl *AuthnHttpClient) RefreshToken(oldToken string) (newToken string, err error) {

	clientID := cl.tlsClient.GetClientID()
	refreshPath := server.HttpPostRefreshPath
	dataJSON, _ := jsoniter.Marshal(oldToken)
	// first initialize the client with the old token
	cl.tlsClient.ConnectWithToken(clientID, oldToken)
	// post a request and expect an instance response
	outputRaw, status, err := cl.tlsClient.Post(refreshPath, dataJSON)

	if err == nil && status == http.StatusOK {
		err = jsoniter.Unmarshal(outputRaw, &newToken)
	}

	if err != nil {
		slog.Warn("RefreshToken failed: " + err.Error())
	} else {
		// apply the token to the connection
		err = cl.tlsClient.ConnectWithToken(clientID, oldToken)
	}
	return newToken, err
}

// NewAuthnHttpClient creates an instance of the authentication client to login and obtain
// auth tokens.
//
//	serverURL is the host:port of the http server
//	caCert is the server CA
func NewAuthnHttpClient(serverURL string, caCert *x509.Certificate) *AuthnHttpClient {
	parts, err := url.Parse(serverURL)
	if err != nil {
		slog.Error("NewAuthnClient: invalid server URL", "err", err.Error())
		return nil
	}

	tlsClient := tlsclient.NewTLSClient(parts.Host, nil, caCert, 0)
	return &AuthnHttpClient{
		tlsClient: tlsClient,
	}
}

// LoginWithForm invokes login using a form - temporary helper
// intended for testing a connection to a web server.
//
// This sets the bearer token for further requests. It requires the server
// to set a session cookie in response to the login.
//func (cl *HiveotSseClient) LoginWithForm(
//	password string) (newToken string, err error) {
//
//	// TBD: does this client need a cookie jar???
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

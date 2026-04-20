package httpauthn

import (
	"io"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/authn"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/utils"
	jsoniter "github.com/json-iterator/go"
)

const (
	// HttpPostLoginPath is the http authentication endpoint of the module
	HttpPostLoginPath   = "/authn/login"
	HttpPostLogoutPath  = "/authn/logout"
	HttpPostRefreshPath = "/authn/refresh"
	HttpGetProfilePath  = "/authn/profile"
)

// AuthnUserHttpService is the module that offers the REST api for handling user requests
// such as login, logout, refresh over http.
//
// This converts http to RRN requests that are handled downstream
type AuthnUserHttpService struct {
	modules.HiveModuleBase
	httpServer transports.IHttpServer
}

// GetConnectURL returns the URI of the authentication server with protocol to include
// in the TD security scheme.
//
// This is currently just the base for the login endpoint (post {base}/authn/login).
// The http server might need to include a web page where users can enter their login
// name and password, although that won't work for machines... tbd
//
// Note that web browsers do not directly access the runtime endpoints.
// Instead a web server (hiveoview or other) provides the user interface.
// Including the auth endpoint here is currently just a hint. How to integrate this?
func (m *AuthnUserHttpService) GetConnectURL() (uri string, protocolType string) {
	baseURL := m.httpServer.GetConnectURL()
	loginURL, _ := url.JoinPath(baseURL, HttpPostLoginPath)
	return loginURL, transports.ProtocolTypeWotHttpBasic
}

// onHttpGetProfile returns the client's profile
func (m *AuthnUserHttpService) onHttpGetProfile(w http.ResponseWriter, r *http.Request) {
	var profile authn.ClientProfile
	rp, err := m.httpServer.GetRequestParams(r)

	if err == nil {
		err = m.Rpc(
			rp.ClientID, td.OpInvokeAction, authn.AuthnUserServiceID,
			authn.UserActionGetProfile, nil, &profile)
	}
	if err != nil {
		slog.Warn("onHttpGetProfile failed", "clientID", rp.ClientID, "err", err.Error())
	}
	utils.WriteReply(w, true, profile, err)
}

// onHttpLogin handles a login request and returns an auth token.
//
// Body contains {"username":name, "password":pass} format
// This is the only unprotected route supported.
// This uses the configured session authenticator.
func (m *AuthnUserHttpService) onHttpLogin(w http.ResponseWriter, r *http.Request) {
	var newToken string
	var args authn.UserLoginArgs

	payload, err := io.ReadAll(r.Body)
	if err == nil {
		err = jsoniter.Unmarshal(payload, &args)
	}
	if err == nil {
		// the login is handled in-house and has an immediate return

		err = m.Rpc(args.UserName,
			td.OpInvokeAction, authn.AuthnUserServiceID, authn.UserActionLogin,
			&args, &newToken)

		slog.Info("onHttpLogin", "clientID", args.UserName)
	}
	if err != nil {
		slog.Warn("onHttpLogin failed:", "err", err.Error())
		utils.WriteError(w, err, http.StatusUnauthorized)
		return
	}
	// TBD: set client session cookie for browser clients
	utils.WriteReply(w, true, newToken, nil)
}

// onHttpLogout ends the session and closes all client connections
func (m *AuthnUserHttpService) onHttpLogout(w http.ResponseWriter, r *http.Request) {
	// use the authenticator
	rp, err := m.httpServer.GetRequestParams(r)

	slog.Info("onHttpLogout", slog.String("clientID", rp.ClientID))
	err = m.Rpc(rp.ClientID,
		td.OpInvokeAction,
		authn.AuthnUserServiceID,
		authn.UserActionLogout, nil, nil)

	utils.WriteReply(w, true, nil, err)
}

// onHttpAuthRefresh refreshes the auth token using the session authenticator.
// The session authenticator is that of the authn service. This allows testing with a dummy
// authenticator without having to run the authn service.
func (m *AuthnUserHttpService) onHttpTokenRefresh(w http.ResponseWriter, r *http.Request) {
	var newToken string
	var oldToken string
	rp, err := m.httpServer.GetRequestParams(r)

	if err == nil {
		jsoniter.Unmarshal(rp.Payload, &oldToken)
		slog.Info("onHttpTokenRefresh", "clientID", rp.ClientID)

		err = m.Rpc(rp.ClientID, td.OpInvokeAction,
			authn.AuthnUserServiceID,
			authn.UserActionRefreshToken, &oldToken, &newToken)
	}
	if err != nil {
		slog.Warn("onHttpTokenRefresh failed:", "err", err.Error())
		utils.WriteError(w, err, 0)
		return
	}
	utils.WriteReply(w, true, newToken, nil)
}

func (m *AuthnUserHttpService) Start() error {
	// create routes
	pubRoutes := m.httpServer.GetPublicRoute()
	pubRoutes.Post(HttpPostLoginPath, m.onHttpLogin)

	protRoutes := m.httpServer.GetProtectedRoute()
	protRoutes.Post(HttpPostRefreshPath, m.onHttpTokenRefresh)
	protRoutes.Post(HttpPostLogoutPath, m.onHttpLogout)
	protRoutes.Get(HttpGetProfilePath, m.onHttpGetProfile)

	return nil
}

func (m *AuthnUserHttpService) Stop() {
	// todo remove registrations
}

// Create an authn handler for serving user requests over http
// This converts http requests to RRN messages that are handled downstream.
func NewAuthnUserHttpService(httpServer transports.IHttpServer) *AuthnUserHttpService {
	if httpServer == nil {
		panic("NewAuthnUserHttpHandler: missing http server")
	}
	handler := &AuthnUserHttpService{
		httpServer: httpServer,
	}
	return handler
}

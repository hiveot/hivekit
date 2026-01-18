package module

import (
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/hiveot/hivekit/go/lib/servers/httpbasic"
	"github.com/hiveot/hivekit/go/lib/servers/tlsserver"
	"github.com/hiveot/hivekit/go/modules/authn"
	jsoniter "github.com/json-iterator/go"
)

// createRoutes adds handlers for authentication methods:
// - login is added to unprotected route
// - refresh, logout is added to the protected route
func (m *AuthnModule) createRoutes() error {
	pubRoutes := m.httpServer.GetPublicRoute()
	pubRoutes.Post(httpbasic.HttpPostLoginPath, m.onHttpLogin)

	protRoutes := m.httpServer.GetProtectedRoute()
	protRoutes.Post(httpbasic.HttpPostRefreshPath, m.onHttpTokenRefresh)
	protRoutes.Post(httpbasic.HttpPostLogoutPath, m.onHttpLogout)
	return nil
}

// onHttpLogin handles a login request and returns an auth token.
//
// Body contains {"login":name, "password":pass} format
// This is the only unprotected route supported.
// This uses the configured session authenticator.
func (m *AuthnModule) onHttpLogin(w http.ResponseWriter, r *http.Request) {
	var newToken string
	var args authn.UserLoginArgs
	var validUntil time.Time

	payload, err := io.ReadAll(r.Body)
	if err == nil {
		err = jsoniter.Unmarshal(payload, &args)
	}
	if err == nil {
		// the login is handled in-house and has an immediate return
		// TODO: use-case for 3rd party login? oauth2 process support? tbd
		// FIXME: hard-coded keys!? ugh
		newToken, validUntil, err = m.Login(args.Login, args.Password)

		_ = validUntil
		slog.Info("HandleLogin", "clientID", args.Login)
	}
	if err != nil {
		slog.Warn("HandleLogin failed:", "err", err.Error())
		tlsserver.WriteError(w, err, http.StatusUnauthorized)
		return
	}
	// TODO: set client session cookie for browser clients
	//srv.sessionManager.SetSessionCookie(cs.sessionID,token)
	tlsserver.WriteReply(w, true, newToken, nil)
}

// onHttpLogout ends the session and closes all client connections
func (m *AuthnModule) onHttpLogout(w http.ResponseWriter, r *http.Request) {
	// use the authenticator
	rp, err := m.httpServer.GetRequestParams(r)
	if err == nil {
		slog.Info("HandleLogout", "clientID", rp.ClientID)
		m.Logout(rp.ClientID)
	}
	tlsserver.WriteReply(w, true, nil, err)
}

// onHttpAuthRefresh refreshes the auth token using the session authenticator.
// The session authenticator is that of the authn service. This allows testing with a dummy
// authenticator without having to run the authn service.
func (m *AuthnModule) onHttpTokenRefresh(w http.ResponseWriter, r *http.Request) {
	var newToken string
	var oldToken string
	var validUntil time.Time
	rp, err := m.httpServer.GetRequestParams(r)

	if err == nil {
		jsoniter.Unmarshal(rp.Payload, &oldToken)
		slog.Info("HandleAuthRefresh", "clientID", rp.ClientID)
		newToken, validUntil, err = m.authenticator.RefreshToken(rp.ClientID, oldToken)
		_ = validUntil
	}
	if err != nil {
		slog.Warn("HandleAuthRefresh failed:", "err", err.Error())
		tlsserver.WriteError(w, err, 0)
		return
	}
	tlsserver.WriteReply(w, true, newToken, nil)
}

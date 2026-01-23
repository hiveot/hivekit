package authnserver

import (
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/hiveot/hivekit/go/lib/servers/httpbasic"
	"github.com/hiveot/hivekit/go/lib/servers/tlsserver"
	"github.com/hiveot/hivekit/go/modules/transports"
	jsoniter "github.com/json-iterator/go"
)

const (
	// HttpPostLoginPath is the http authentication endpoint of the module
	HttpPostLoginPath   = "/authn/login"
	HttpPostLogoutPath  = "/authn/logout"
	HttpPostRefreshPath = "/authn/refresh"
)

// helper for building a login request message
// used in http and rrn messaging
type UserLoginArgs struct {
	UserName string `json:"username"`
	Password string `json:"password"`
}

// UserHttpHandler for handling user requests such as login, logout, refresh over http
type UserHttpHandler struct {
	// module authn.IAuthn
	authenticator transports.IAuthenticator
	httpServer    transports.IHttpServer
}

// onHttpLogin handles a login request and returns an auth token.
//
// Body contains {"username":name, "password":pass} format
// This is the only unprotected route supported.
// This uses the configured session authenticator.
func (handler *UserHttpHandler) onHttpLogin(w http.ResponseWriter, r *http.Request) {
	var newToken string
	var args UserLoginArgs
	var validUntil time.Time

	payload, err := io.ReadAll(r.Body)
	if err == nil {
		err = jsoniter.Unmarshal(payload, &args)
	}
	if err == nil {
		// the login is handled in-house and has an immediate return
		// TODO: use-case for 3rd party login? oauth2 process support? tbd
		// FIXME: hard-coded keys!? ugh
		newToken, validUntil, err = handler.authenticator.Login(args.UserName, args.Password)

		_ = validUntil
		slog.Info("HandleLogin", "clientID", args.UserName)
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
func (handler *UserHttpHandler) onHttpLogout(w http.ResponseWriter, r *http.Request) {
	// use the authenticator
	rp, err := handler.httpServer.GetRequestParams(r)
	if err == nil {
		slog.Info("HandleLogout", "clientID", rp.ClientID)
		handler.authenticator.Logout(rp.ClientID)
	}
	tlsserver.WriteReply(w, true, nil, err)
}

// onHttpAuthRefresh refreshes the auth token using the session authenticator.
// The session authenticator is that of the authn service. This allows testing with a dummy
// authenticator without having to run the authn service.
func (handler *UserHttpHandler) onHttpTokenRefresh(w http.ResponseWriter, r *http.Request) {
	var newToken string
	var oldToken string
	var validUntil time.Time
	rp, err := handler.httpServer.GetRequestParams(r)

	if err == nil {
		jsoniter.Unmarshal(rp.Payload, &oldToken)
		slog.Info("HandleAuthRefresh", "clientID", rp.ClientID)
		newToken, validUntil, err = handler.authenticator.RefreshToken(rp.ClientID, oldToken)
		_ = validUntil
	}
	if err != nil {
		slog.Warn("HandleAuthRefresh failed:", "err", err.Error())
		tlsserver.WriteError(w, err, 0)
		return
	}
	tlsserver.WriteReply(w, true, newToken, nil)
}

// Create a http server handler for user facing requests and register endpoints
func NewUserHttpHandler(authenticator transports.IAuthenticator, httpServer transports.IHttpServer) *UserHttpHandler {
	handler := &UserHttpHandler{
		httpServer:    httpServer,
		authenticator: authenticator,
	}
	// create routes
	pubRoutes := httpServer.GetPublicRoute()
	pubRoutes.Post(httpbasic.HttpPostLoginPath, handler.onHttpLogin)

	protRoutes := httpServer.GetProtectedRoute()
	protRoutes.Post(httpbasic.HttpPostRefreshPath, handler.onHttpTokenRefresh)
	protRoutes.Post(httpbasic.HttpPostLogoutPath, handler.onHttpLogout)
	return handler
}

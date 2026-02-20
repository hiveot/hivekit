package server

import (
	"io"
	"log/slog"
	"net/http"
	"time"

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

// helper for building a login request message
// used in http and rrn messaging
type UserLoginArgs struct {
	UserName string `json:"username"`
	Password string `json:"password"`
}

// UserHttpHandler for handling user requests such as login, logout, refresh over http
type UserHttpHandler struct {
	m          authn.IAuthnModule
	httpServer transports.IHttpServer
}

// onHttpGetProfile returns the client's profile
func (handler *UserHttpHandler) onHttpGetProfile(w http.ResponseWriter, r *http.Request) {
	var profile authn.ClientProfile
	rp, err := handler.httpServer.GetRequestParams(r)
	if err == nil {
		profile, err = handler.m.GetProfile(rp.ClientID)
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
		newToken, validUntil, err = handler.m.Login(args.UserName, args.Password)

		_ = validUntil
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
func (handler *UserHttpHandler) onHttpLogout(w http.ResponseWriter, r *http.Request) {
	// use the authenticator
	rp, err := handler.httpServer.GetRequestParams(r)
	if err == nil {
		slog.Info("onHttpLogout", "clientID", rp.ClientID)
		handler.m.Logout(rp.ClientID)
	}
	utils.WriteReply(w, true, nil, err)
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
		slog.Info("onHttpTokenRefresh", "clientID", rp.ClientID)
		newToken, validUntil, err = handler.m.RefreshToken(rp.ClientID, oldToken)
		_ = validUntil
	}
	if err != nil {
		slog.Warn("onHttpTokenRefresh failed:", "err", err.Error())
		utils.WriteError(w, err, 0)
		return
	}
	utils.WriteReply(w, true, newToken, nil)
}

// Create a http server handler for user facing requests and register endpoints
func NewUserHttpHandler(m authn.IAuthnModule, httpServer transports.IHttpServer) *UserHttpHandler {
	if m == nil || httpServer == nil {
		panic("NewUserHttpHandler: nil parameter")
	}
	handler := &UserHttpHandler{
		httpServer: httpServer,
		m:          m,
	}
	// create routes
	pubRoutes := httpServer.GetPublicRoute()
	pubRoutes.Post(HttpPostLoginPath, handler.onHttpLogin)

	protRoutes := httpServer.GetProtectedRoute()
	protRoutes.Post(HttpPostRefreshPath, handler.onHttpTokenRefresh)
	protRoutes.Post(HttpPostLogoutPath, handler.onHttpLogout)
	protRoutes.Get(HttpGetProfilePath, handler.onHttpGetProfile)
	return handler
}

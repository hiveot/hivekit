package module

import (
	"fmt"
	"slices"
	"time"

	"github.com/hiveot/hivekit/go/lib/servers/tlsserver"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/modules/transports/httpbasic"
	"github.com/hiveot/hivekit/go/msg"
	"github.com/hiveot/hivekit/go/utils"
	jsoniter "github.com/json-iterator/go"

	"io"
	"log/slog"
	"net/http"
)

// createRoutes creates the middleware chain for handling requests, including
// recoverer, compression and token verification for protected routes.
//
// This includes the unprotected routes for login and ping (for now)
// This includes the protected routes for refresh and logout. (for now)
// Everything else should be added by the sub-protocols.
//
// Routes are added by (sub)protocols such as http-basic, sse and wss.
func (m *HttpBasicModule) createRoutes() {

	// TODO: add csrf support in posts
	//csrfMiddleware := csrf.Protect(
	//	[]byte("32-byte-long-auth-key"),
	//	csrf.SameSite(csrf.SameSiteStrictMode))

	//-- add the middleware before routes
	// router.Use(middleware.Recoverer)
	//router.Use(middleware.Logger) // todo: proper logging strategy
	//router.Use(csrfMiddleware)
	// router.Use(middleware.Compress(5,
	// "text/html", "text/css", "text/javascript", "image/svg+xml"))

	//--- public routes do not require an authenticated session
	pubRoutes := m.httpServer.GetPublicRoute()

	//r.Get("/static/*", staticFileServer.ServeHTTP)
	// build-in REST API for easy login to obtain a token

	// register authentication endpoints
	// FIXME: determine how WoT wants auth endpoints to be published
	pubRoutes.Post(httpbasic.HttpPostLoginPath, m.onHttpLogin)
	pubRoutes.Get(httpbasic.HttpGetPingPath, m.onHttpPing)

	//--- private routes that requires authentication (as published in the TD)
	protRoutes := m.httpServer.GetProtectedRoute()
	// client sessions authenticate the sender
	// protRoutes.Use(AddSessionFromToken(srv.authenticator))

	// sub-protocols can add protected routes
	// srv.protectedRoutes = r

	// register generic handlers for operations on Thing and affordance level
	// these endpoints are published in the forms of each TD. See also AddTDForms.
	protRoutes.HandleFunc(httpbasic.HttpBasicAffordanceOperationPath, m.onHttpAffordanceOperation)
	protRoutes.HandleFunc(httpbasic.HttpBasicThingOperationPath, m.onHttpThingOperation)

	// http supported authentication endpoints
	protRoutes.Post(httpbasic.HttpPostRefreshPath, m.onHttpAuthRefresh)
	protRoutes.Post(httpbasic.HttpPostLogoutPath, m.onHttpLogout)
}

// EnableStatic adds a path to read files from the static directory. Auth required.
//
//	base is the base path on which to serve the static files, eg: "/static"
//	staticRoot is the root directory where static files are kept. This must be a full path.
func (m *HttpBasicModule) EnableStatic(base string, staticRoot string) error {
	protRoutes := m.httpServer.GetProtectedRoute()
	if protRoutes == nil || base == "" {
		return fmt.Errorf("no protected route or invalid parameters")
	}
	var staticFileServer http.Handler
	// if staticRoot == "" {
	// 	staticFileServer = http.FileServer(
	// 		&StaticFSWrapper{
	// 			FileSystem:   http.FS(src.EmbeddedStatic),
	// 			FixedModTime: time.Now(),
	// 		})
	// } else {
	// during development when run from the 'hub' project directory
	staticFileServer = http.FileServer(http.Dir(staticRoot))
	// }
	staticPath := base + "/*"
	protRoutes.Get(staticPath, staticFileServer.ServeHTTP)
	return nil
}

// onHttpAffordanceOperation converts the http request to a request message and pass it to the
// registered request handler.
// This read request params for {operation}, {thingID} and {name}
func (m *HttpBasicModule) onHttpAffordanceOperation(w http.ResponseWriter, r *http.Request) {
	var output any
	var handled bool

	// 1. Decode the request message
	rp, err := m.httpServer.GetRequestParams(r)
	if err != nil {
		slog.Error(err.Error())
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	// Use the authenticated clientID as the sender
	var input any
	err = jsoniter.Unmarshal(rp.Payload, &input)
	req := msg.NewRequestMessage(rp.Op, rp.ThingID, rp.Name, input, "")
	req.SenderID = rp.ClientID
	req.CorrelationID = rp.CorrelationID

	// filter on allowed operations
	if !slices.Contains(HttpKnownOperations, req.Operation) {
		slog.Warn("Unsupported operation for http-basic",
			"method", r.Method, "URL", r.URL.String(),
			"operation", req.Operation, "thingID", req.ThingID, "name", req.Name, "clientID", req.SenderID)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// This passes the request to the handler provided on startup. The replyTo is
	// expected to be called before the timeout, otherwise this returns an error.
	rx := utils.NewAsyncReceiver[*msg.ResponseMessage]()
	err = m.serverRequestHandler(req, func(resp *msg.ResponseMessage) error {
		if resp != nil {
			if resp.Error != nil {
				err = resp.Error.AsError()
			}
		}
		rx.SetResponse(resp, err)
		return nil
	})
	resp, err := rx.WaitForResponse(time.Second * 1)
	if resp != nil {
		output = resp.Value
	} else {
		slog.Info("no response")
	}

	// 4. Return the response
	tlsserver.WriteReply(w, handled, output, err)
}

// onHttpThingOperation converts the http request to a request message and pass it to the registered request handler
func (m *HttpBasicModule) onHttpThingOperation(w http.ResponseWriter, r *http.Request) {
	// same same
	m.onHttpAffordanceOperation(w, r)
}

// onHttpLogin handles a login request and returns an auth token.
//
// Body contains {"login":name, "password":pass} format
// This is the only unprotected route supported.
// This uses the configured session authenticator.
func (m *HttpBasicModule) onHttpLogin(w http.ResponseWriter, r *http.Request) {
	var reply any
	var args transports.UserLoginArgs

	payload, err := io.ReadAll(r.Body)
	if err == nil {
		err = jsoniter.Unmarshal(payload, &args)
	}
	if err == nil {
		// the login is handled in-house and has an immediate return
		// TODO: use-case for 3rd party login? oauth2 process support? tbd
		// FIXME: hard-coded keys!? ugh
		reply, err = m.authenticator.Login(args.Login, args.Password)
		slog.Info("HandleLogin", "clientID", args.Login)
	}
	if err != nil {
		slog.Warn("HandleLogin failed:", "err", err.Error())
		tlsserver.WriteError(w, err, http.StatusUnauthorized)
		return
	}
	// TODO: set client session cookie for browser clients
	//srv.sessionManager.SetSessionCookie(cs.sessionID,token)
	tlsserver.WriteReply(w, true, reply, nil)
}

// onHttpAuthRefresh refreshes the auth token using the session authenticator.
// The session authenticator is that of the authn service. This allows testing with a dummy
// authenticator without having to run the authn service.
func (m *HttpBasicModule) onHttpAuthRefresh(w http.ResponseWriter, r *http.Request) {
	var newToken string
	var oldToken string
	rp, err := m.httpServer.GetRequestParams(r)

	if err == nil {
		jsoniter.Unmarshal(rp.Payload, &oldToken)
		slog.Info("HandleAuthRefresh", "clientID", rp.ClientID)
		newToken, err = m.authenticator.RefreshToken(rp.ClientID, oldToken)
	}
	if err != nil {
		slog.Warn("HandleAuthRefresh failed:", "err", err.Error())
		tlsserver.WriteError(w, err, 0)
		return
	}
	tlsserver.WriteReply(w, true, newToken, nil)
}

// onHttpLogout ends the session and closes all client connections
func (m *HttpBasicModule) onHttpLogout(w http.ResponseWriter, r *http.Request) {
	// use the authenticator
	rp, err := m.httpServer.GetRequestParams(r)
	if err == nil {
		slog.Info("HandleLogout", "clientID", rp.ClientID)
		m.authenticator.Logout(rp.ClientID)
	}
	tlsserver.WriteReply(w, true, nil, err)
}

// onHttpPing with http handler returns a pong response
func (m *HttpBasicModule) onHttpPing(w http.ResponseWriter, r *http.Request) {
	// simply return a pong message
	tlsserver.WriteReply(w, true, "pong", nil)
}

package module

import (
	"fmt"
	"slices"
	"time"

	"github.com/hiveot/hivekit/go/modules/transports/httpbasic"
	"github.com/hiveot/hivekit/go/msg"
	"github.com/hiveot/hivekit/go/utils"
	jsoniter "github.com/json-iterator/go"

	"log/slog"
	"net/http"
)

// createRoutes creates the middleware chain for handling requests, including
// recoverer, compression and token verification for protected routes.
//
// This includes the unprotected routes for ping (for now)
// Everything else should be added by the sub-protocols.
//
// Routes are added by (sub)protocols such as http-basic, sse and wss.
func (m *HttpBasicModule) createRoutes() {

	//--- public routes do not require an authenticated session
	pubRoutes := m.httpServer.GetPublicRoute()

	//r.Get("/static/*", staticFileServer.ServeHTTP)
	// build-in REST API for easy login to obtain a token

	// register authentication endpoints
	// FIXME: determine how WoT wants auth endpoints to be published
	// pubRoutes.Post(httpbasic.HttpPostLoginPath, m.onHttpLogin)
	pubRoutes.Get(httpbasic.HttpGetPingPath, m.onHttpPing)

	//--- private routes that requires authentication (as published in the TD)
	protRoutes := m.httpServer.GetProtectedRoute()
	if protRoutes == nil {
		panic("no protected route available")
	}
	// client sessions authenticate the sender
	// protRoutes.Use(AddSessionFromToken(srv.authenticator))

	// sub-protocols can add protected routes
	// srv.protectedRoutes = r

	// register generic handlers for operations on Thing and affordance level
	// these endpoints are published in the forms of each TD. See also AddTDForms.
	protRoutes.HandleFunc(httpbasic.HttpBasicAffordanceOperationPath, m.onHttpAffordanceOperation)
	protRoutes.HandleFunc(httpbasic.HttpBasicThingOperationPath, m.onHttpThingOperation)

	// http supported authentication endpoints
	// protRoutes.Post(httpbasic.HttpPostRefreshPath, m.onHttpAuthRefresh)
	// protRoutes.Post(httpbasic.HttpPostLogoutPath, m.onHttpLogout)
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
		var err2 error
		if resp != nil {
			if resp.Error != nil {
				err2 = resp.Error.AsError()
			}
		}
		rx.SetResponse(resp, err2)
		return nil
	})
	resp, err := rx.WaitForResponse(time.Second * 1)
	if resp != nil {
		output = resp.Value
	} else {
		slog.Info("no response")
	}

	// 4. Return the response
	utils.WriteReply(w, handled, output, err)
}

// onHttpThingOperation converts the http request to a request message and pass it to the registered request handler
func (m *HttpBasicModule) onHttpThingOperation(w http.ResponseWriter, r *http.Request) {
	// same same
	m.onHttpAffordanceOperation(w, r)
}

// onHttpPing with http handler returns a pong response
func (m *HttpBasicModule) onHttpPing(w http.ResponseWriter, r *http.Request) {
	// simply return a pong message
	utils.WriteReply(w, true, "pong", nil)
}

package httpbasicserver

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
func (m *HttpBasicServer) createRoutes() {

	//--- public routes do not require an authenticated session
	pubRoutes := m.httpServer.GetPublicRoute()
	_ = pubRoutes

	//pubRoutes.Get("/static/*", staticFileServer.ServeHTTP)

	//--- private routes that requires authentication (as published in the TD)
	protRoutes := m.httpServer.GetProtectedRoute()
	if protRoutes == nil {
		panic("no protected route available")
	}

	// register generic handlers for operations on Thing and affordance level
	// these endpoints are published in the forms of each TD. See also AddTDForms.
	protRoutes.HandleFunc(httpbasic.HttpBasicAffordanceOperationPath, m.onHttpAffordanceOperation)
	protRoutes.HandleFunc(httpbasic.HttpBasicThingOperationPath, m.onHttpThingOperation)

}

// EnableStatic adds a path to read files from the static directory. Auth required.
//
//	base is the base path on which to serve the static files, eg: "/static"
//	staticRoot is the root directory where static files are kept. This must be a full path.
func (m *HttpBasicServer) EnableStatic(base string, staticRoot string) error {
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
func (m *HttpBasicServer) onHttpAffordanceOperation(w http.ResponseWriter, r *http.Request) {
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
		rx.SetResponse(resp)
		return nil
	})
	resp, err := rx.WaitForResponse(time.Second * 1)
	if resp != nil {
		output = resp.Output
	} else {
		slog.Info("no response")
	}

	// 4. Return the response
	utils.WriteReply(w, handled, output, err)
}

// onHttpThingOperation converts the http request to a request message and pass it to the registered request handler
func (m *HttpBasicServer) onHttpThingOperation(w http.ResponseWriter, r *http.Request) {
	// same same
	m.onHttpAffordanceOperation(w, r)
}

// onHttpPing with http handler returns a pong response
func (m *HttpBasicServer) onHttpPing(w http.ResponseWriter, r *http.Request) {
	// simply return a pong message
	utils.WriteReply(w, true, "pong", nil)
}

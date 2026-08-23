package serverimpl

import (
	"context"
	"fmt"
	"slices"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/cells/transport/httpbasic"
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
// The http-basic routes include generic routes for affordances using URI variables for
// handling a single endpoint for all operations. See HttpBasicThingOperationPath and
// HttpBasicAffordanceOperationPath for the generic paths.
// The paths are included in the thing level forms when invoking AddTDSecForms.
func (srv *HttpBasicServerImpl) createRoutes() {

	//--- public routes do not require an authenticated session
	pubRoutes := srv.httpServer.GetPublicRoute()
	_ = pubRoutes

	//pubRoutes.Get("/static/*", staticFileServer.ServeHTTP)

	//--- private routes that requires authentication (as published in the TD)
	protRoutes := srv.httpServer.GetProtectedRoute()
	if protRoutes == nil {
		panic("no protected route available")
	}

	// register generic handlers for operations on Thing and affordance level
	// these endpoints are published in the forms of each TD. See also AddTDForms.

	protRoutes.HandleFunc(
		httpbasic.HttpBasicNotificationPath, srv.onHttpNotification)
	protRoutes.HandleFunc(
		httpbasic.HttpBasicAffordanceOperationPath, srv.onHttpAffordanceOperation)
	protRoutes.HandleFunc(
		httpbasic.HttpBasicThingOperationPath, srv.onHttpThingOperation)

}

// EnableStatic adds a path to read files from the static directory. Auth required.
//
//	base is the base path on which to serve the static files, eg: "/static"
//	staticRoot is the root directory where static files are kept. This must be a full path.
func (srv *HttpBasicServerImpl) EnableStatic(base string, staticRoot string) error {
	protRoutes := srv.httpServer.GetProtectedRoute()
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

// onHttpNotification converts the http request to a notification message and forward
// it to the notification sink
func (srv *HttpBasicServerImpl) onHttpNotification(w http.ResponseWriter, r *http.Request) {

	// 1. Decode the http request
	rp, err := srv.httpServer.GetRequestParams(r)
	if err != nil {
		slog.Error(err.Error())
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	var notif *msg.NotificationMessage
	if len(rp.Payload) > 0 {
		err = jsoniter.Unmarshal(rp.Payload, &notif)
	}

	// Use the authenticated clientID as the sender
	if err != nil {
		utils.WriteError(w, err, http.StatusBadRequest)
		return
	}
	notif.SenderID = rp.ClientID
	srv.ForwardNotification(notif)
	utils.WriteReply(w, true, nil, nil)
}

// onHttpAffordanceOperation converts the http request to a request message and pass it to the
// registered request handler.
//
// This also handles notifications for hiveot compatibility sakes.
//
// This read request params for {op}, {id} and {name}
func (srv *HttpBasicServerImpl) onHttpAffordanceOperation(w http.ResponseWriter, r *http.Request) {
	var output any
	var handled bool

	// 1. Decode the request message
	rp, err := srv.httpServer.GetRequestParams(r)
	if err != nil {
		slog.Error(err.Error())
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	// Use the authenticated clientID as the sender
	var input any
	if len(rp.Payload) > 0 {
		err = jsoniter.Unmarshal(rp.Payload, &input)
	}

	// construct a request message
	req := msg.NewRequestMessage(rp.Op, rp.ThingID, rp.Name, input)
	req.CorrelationID = rp.CorrelationID
	req.SenderID = rp.ClientID

	// filter on allowed operations
	if slices.Contains(thingLevelOperations, req.Operation) {
	} else if slices.Contains(affordanceOperations, req.Operation) {
	} else {
		slog.Warn("Unsupported operation for http-basic",
			"method", r.Method, "URL", r.URL.String(),
			"operation", req.Operation, "thingID", req.ThingID, "name", req.Name, "clientID", req.SenderID)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// handle ping operation
	if req.Operation == td.HTOpPing {
		utils.WriteReply(w, true, req.Input, err)
		return
	}

	// This passes the request to the request sink. The replyTo is expected to be called
	// before the timeout, otherwise this returns an error.
	ctx, cancelFn := context.WithTimeout(context.Background(), srv.GetTimeout())
	rx := utils.NewAsyncReceiver[*msg.ResponseMessage]()
	err = srv.ForwardRequest(req, func(resp *msg.ResponseMessage) error {
		rx.SetResponse(resp)
		cancelFn()
		return nil
	})
	<-ctx.Done()
	resp, err := rx.WaitForResponse(0)
	if resp != nil {
		output = resp.Output
	} else {
		slog.Info("no response")
	}

	// 4. Return the response
	utils.WriteReply(w, handled, output, err)
}

// onHttpThingOperation converts the http request to a request message and pass it to the registered request handler
func (srv *HttpBasicServerImpl) onHttpThingOperation(w http.ResponseWriter, r *http.Request) {
	// same same
	srv.onHttpAffordanceOperation(w, r)
}

// onHttpPing with http handler returns a pong response
func (srv *HttpBasicServerImpl) onHttpPing(w http.ResponseWriter, r *http.Request) {
	// simply return a pong message
	utils.WriteReply(w, true, "pong", nil)
}

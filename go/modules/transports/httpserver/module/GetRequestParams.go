package module

import (
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hivekit/go/modules/transports/httpserver"
)

// GetClientIdFromContext returns the authenticated clientID for the given request
func GetClientIdFromContext(r *http.Request) (clientID string, err error) {
	ctxClientID := r.Context().Value(httpserver.ClientContextID)
	if ctxClientID == nil {
		return "", errors.New("no clientID in context")
	}
	clientID = ctxClientID.(string)
	return clientID, nil
}

// GetRequestParams reads the client session, URL parameters and body payload from the
// http request context.
//
// The session context is set by the http middleware. If the session is not available then
// this returns an error. Note that the session middleware handler will block any request
// that requires a session.
//
// This protocol binding determines three variables, {thingID}, {name} and {op} from the path.
// It unmarshal's the request body into 'data', if given.
//
//	{operation} is the operation
//	{thingID} is the agent or digital twin thing ID
//	{name} is the property, event or action name. '+' means 'all'
func GetRequestParams(r *http.Request) (reqParam httpserver.RequestParams, err error) {

	// get the required client session of this agent
	reqParam.ClientID, err = GetClientIdFromContext(r)
	if err != nil {
		// This is an internal error. The middleware session handler would have blocked
		// a request that required a session before getting here.
		slog.Error(err.Error())
		return reqParam, err
	}
	correlationID := r.Header.Get(httpserver.CorrelationIDHeader)
	reqParam.CorrelationID = correlationID

	// A connection ID distinguishes between different connections from the same client.
	// This is used to correlate http requests with out-of-band responses like a SSE
	// return channel.
	// If a 'cid' header exists, use it as the connection ID.
	headerCID := r.Header.Get(httpserver.ConnectionIDHeader)
	if headerCID == "" {
		// FIXME: this is only an issue with hiveot-sse. Maybe time to retire it?
		// alt: use a session-id from the auth token - two browser connections would
		// share this however.

		// http-basic isn't be bothered. Each WoT sse connection is the subscription
		//  (only a single subscription per sse connection which is nearly useless)
		slog.Info("GetRequestParams: missing connection-id, only a single " +
			"connection is supported")
	}

	reqParam.ConnectionID = headerCID

	// URLParam names must match the  path variables set in the router.
	reqParam.ThingID = chi.URLParam(r, httpserver.ThingIDURIVar)
	reqParam.Name = chi.URLParam(r, httpserver.NameURIVar)
	reqParam.Op = chi.URLParam(r, httpserver.OperationURIVar)
	if r.Body != nil {
		reqParam.Payload, _ = io.ReadAll(r.Body)
	}

	return reqParam, err
}

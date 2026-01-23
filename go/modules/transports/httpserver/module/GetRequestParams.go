package module

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hivekit/go/modules/transports"
)

// GetClientIdFromContext returns the authenticated clientID for the given request
func GetClientIdFromContext(r *http.Request) (clientID string, err error) {
	ctxClientID := r.Context().Value(transports.ClientContextID)
	if ctxClientID == nil {
		return "", errors.New("no clientID in context")
	}
	clientID = ctxClientID.(string)
	return clientID, nil
}

// GetRequestParams reads the client session, URL parameters and body payload from the
// http request context.
//
// The session context is set by the http middleware.
// This first checks for a clientID from the session context, which gets it from
// the bearer token auth.
// If no clientID is available but a client certificate is available then use its
// common name (cn) the clientID.
// If the clientID is not available then this returns an error.
//
// This determines {thingID}, {name} and {op} from the path.
// It unmarshals the request body into 'data', if given.
//
//	{operation} is the operation
//	{thingID} is the agent or digital twin thing ID
//	{name} is the property, event or action name. '+' means 'all'
func GetRequestParams(r *http.Request) (reqParam transports.RequestParams, err error) {
	// determine the clientID, either from context or client cert
	reqParam.ClientID, err = GetClientIdFromContext(r)
	if err != nil {
		clcerts := r.TLS.PeerCertificates
		if len(clcerts) > 0 {
			clientCert := clcerts[0]
			reqParam.ClientID = clientCert.Subject.CommonName
		}
	}
	if reqParam.ClientID == "" {
		err := fmt.Errorf("Missing clientID")
		slog.Error(err.Error())
		return reqParam, err
	}
	err = nil
	correlationID := r.Header.Get(transports.CorrelationIDHeader)
	reqParam.CorrelationID = correlationID

	// A connection ID distinguishes between different connections from the same client.
	// This is used to correlate http requests with out-of-band responses like a SSE
	// return channel.
	// If a 'cid' header exists, use it as the connection ID.
	headerCID := r.Header.Get(transports.ConnectionIDHeader)
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
	reqParam.ThingID = chi.URLParam(r, transports.ThingIDURIVar)
	reqParam.Name = chi.URLParam(r, transports.NameURIVar)
	reqParam.Op = chi.URLParam(r, transports.OperationURIVar)
	if r.Body != nil {
		reqParam.Payload, _ = io.ReadAll(r.Body)
	}

	return reqParam, err
}

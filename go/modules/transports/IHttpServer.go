// Package transports with http server and TLSClient apis
package transports

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Standardized http transport definitions for use with HiveOT.
// These are used by GetRequestParams but usage is optional.
const (
	// The http server module type that can be used to retrieve the server instance from the factor.
	HttpServerModuleType = "httpserver"

	// The default http server module instance ID
	DefaultHttpServerModuleID = "httpserver"

	// The default health check ping path to register
	DefaultPingPath = "/ping"

	// The default HTTP TLS listening port if none is set
	DefaultHttpsPort = 8444

	// The context ID's for authenticated clientID and connectionID
	ClientIDContextID = "client-id"
	// The client provided connection ID to differentiate different connections from the same clientID
	ClientCIDContextID = "cid"

	// context for session identification - not currently in use
	// SessionContextID = "sessionID"

	// ConnectionIDHeader is intended for linking return channels to requests.
	// intended for separated return channel like sse.
	ConnectionIDHeader = "cid"
	// CorrelationIDHeader is the header to be able to link requests to out of band responses
	// tentative as it isn't part of the wot spec
	CorrelationIDHeader = "correlationID"

	// URI variables for use in paths. These are read in GetRequestParameters
	// usage is for convenience.
	OperationURIVar = "operation"
	ThingIDURIVar   = "thingID"
	NameURIVar      = "name"
)

// RequestParams contains the parameters read from the HTTP request
type RequestParams struct {
	ClientID      string // authenticated client ID
	CorrelationID string // tentative as it isn't in the spec
	ConnectionID  string // connectionID as provided by the client
	Payload       []byte // the raw request payload (body)

	// optional URI parameters
	ThingID string // the thing ID if defined in the URL as {thingID}
	Name    string // the affordance name if defined in the URL as {name}
	Op      string // the operation if defined in the URL as {op}
}

// IHttpServer is the minimal HTTP server interface as used by various http subprotocols.
// The subprotocols can work with any http server module that supports this interface.
// The factory provides a GetHttpServer() method to retrieve the embedded http server.
type IHttpServer interface {
	// GetAuthenticator returns the authenticator used to authenticate incoming connections
	// Also used by sub-protocols to include security scheme in TD's
	GetAuthenticator() IAuthenticator

	// Returns the connection URL of the http server
	GetConnectURL() string

	// Return the authenticated client ID from the http request context.
	// The clientID are set in the context by the middleware chain.
	GetClientIdFromContext(r *http.Request) (clientID string, err error)

	// GetRequestParams decode the HiveOT standardized request parameters:
	// - clientID from context, provided by 'clientID' context, set by the http server authentication.
	// - connectionID from the 'cid' header
	// - correlationID from the 'correlationID' header
	// - payload from the message body
	// - thingID, operation, name from URI variables
	GetRequestParams(r *http.Request) (RequestParams, error)

	// Return the protected route for adding endpoints.
	// Note that these routes will refuse all requests until an authenticator is configured using
	// SetAuthenticator.
	GetProtectedRoute() chi.Router

	// Return the public route for adding endpoints.
	GetPublicRoute() chi.Router

	// Set the authenticator for http requests
	// This enables the protected routes.
	//
	// Note that the authn module can provide this capability
	SetAuthenticator(authenticator IAuthenticator)

	// Start the server and open the listening port
	Start() error

	// Stop the server and end listening
	Stop()
}

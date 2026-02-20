// Package transports with http server and TLSClient apis
package transports

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"golang.org/x/net/http2"
)

// Standardized http transport definitions for use with HiveOT.
// These are used by GetRequestParams but usage is optional.
const (
	DefaultHttpServerModuleID = "httpserver"

	// The default health check ping path to register
	DefaultPingPath = "/ping"

	// The default listening port if none is set
	DefaultPort = 8444

	// The context ID's for authenticated clientID
	ClientIDContextID = "clientID"

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
type IHttpServer interface {

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
	// Note that these routes will refuse all requests until a validator is configured.
	GetProtectedRoute() chi.Router

	// Return the public route for adding endpoints.
	GetPublicRoute() chi.Router

	// Set the validator for http requests
	// This enables the protected routes
	SetAuthValidator(v ValidateTokenHandler)
}

// ITlsClient interface for generic http/tls client.
//
// Intended to handle the boilerplate for use by http base protocols such as
// WoT http-basic, sse and wss.
//
// The client implementation must handle the certificates for use with TLS.
//
// See also the httpapi.TlsClient which implements this interface.
//
// Most protocol clients simply use this interface which allows the use of a
// replacement implementation.
type ITlsClient interface {

	// Close the connection and release resources
	Close()

	// Connect the client to a server with the clientID and token.
	//
	// If subprotocols require a connection then this will establish that connection.
	// This creates a unque connectionID for the header and places the token in
	// the authorization hedaer.
	ConnectWithToken(clientID string, token string) error

	// Create a new http request with all the headers including authorization.
	// The request can be cancelled using the provided context.
	CreateRequest(ctx context.Context,
		method string, path string, qParams map[string]string,
		body []byte, contentType string,
	) *http.Request

	// Delete is the http convenience function to delete a resource
	Delete(path string) (statusCode int, err error)

	// GetTlsTransport returns the network connection used by this client
	// Intended for use with websocket dialers.
	GetTlsTransport() *http2.Transport

	// GetClientID returns the clientID this client is authenticated as.
	GetClientID() string

	// GetConnectionID returns the unique connection ID for this client as included in the cid header
	GetConnectionID() string

	// GetClientID returns the clientID this client is authenticated as.
	GetHostPort() string

	// GetHttpClient returns the native http client
	// Could be needed in some subprotocols like sse.
	GetHttpClient() *http.Client

	// Get is the http convenience function to retrieve a resource
	Get(path string) (resp []byte, httpStatus int, err error)

	// Connect performs a http connect request (for proxies)
	HttpConnect() (status int, err error)

	// Connect performs a http head request
	Head(path string) (status int, err error)

	// Patch is the http convenience function to partially update a resource
	Patch(path string, body []byte) (output []byte, statusCode int, err error)

	// Ping the well-known "/ping" endpoint on the server for a health check
	Ping() (statusCode int, err error)

	// Post is the http convenience function to create a resource
	Post(path string, body []byte) (output []byte, statusCode int, err error)

	// Postform is the http convenience function to create a resource using http form data
	PostForm(path string, formData map[string]string) (resp []byte, statusCode int, err error)

	// Post is the http convenience function to create or update a resource
	Put(path string, body []byte) (resp []byte, statusCode int, err error)

	// Send an http request.
	//
	// This creates a request object with CreateRequest and a context with timeout,
	// and calls httpClient.Do()
	//
	//	context for cancel function or timeout
	// 	method is one of HTTP's GET|POST|PUT|DELETE|PATCH|...
	//	path is the URL path
	//	qParams are optional query parameters. Use nil to ignore.
	//	body is the optional body to include. Use nil to ignore.
	//	contentType is the optional content-type. Default is "application/json".
	Send(ctx context.Context, method string, path string, qParams map[string]string,
		body []byte, contentType string) (
		resp []byte, httpStatus int, headers http.Header, err error)

	// Change the default timeout for http request to the given timeout
	SetTimeout(timeout time.Duration)

	// Trace performs a message loopback testPost is the http convenience function to create or update a resource
	Trace(path string) (status int, err error)
}

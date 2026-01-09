package httptransport

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/hiveot/hivekit/go/modules"
	"golang.org/x/net/http2"
)

// Standardized http server (and client) constants for use with HiveOT
// These are used by GetRequestParams but usage is optional.
const (
	DefaultHttpServerModuleID = "httpserver"

	// The default listening port if none is set
	DefaultPort = 8444

	// The context ID's for authenticated clientID and sessionID
	ClientContextID  = "clientID"
	SessionContextID = "sessionID"

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

// Configuration options for the https server
type HttpServerConfig struct {
	Address    string            `yaml:"address,omitempty"`
	Port       int               `yaml:"port,omitempty"`
	CaCert     *x509.Certificate `yaml:"-"`
	ServerCert *tls.Certificate  `yaml:"-"`

	// NoTLS disables the use of TLS. For testing obviously
	NoTLS bool

	// Authenticate requests on the protected route.
	// This defaults to nil which means the application add its own handler to the
	// protected route.
	// This handler validates authentication credentials and returns a clientID,
	// sessionID or an error.
	// The server adds them to the request context as ClientContextID and SessionContextID.
	// If an error is returned however, the request ends with http.StatusUnAuthorized response.
	Authenticate func(req *http.Request) (clientID string, sessionID string, err error)

	// CorsEnabled enables the use of net/http CORS and adds the relevant CORS
	// headers to allow browser cross-domain calls in scripts.
	//
	// Use CorsAllowedOrigins to set the additional allowed cross-domains requests.
	//
	// This manages which origins are allowed for example to retrieve data
	// from a different API, as is typical in an IoT environment.
	// Enable this when serving a web site for browsers and allow cross-domain access to
	// specific endpoints on another server.
	// Typically not needed in an IoT setup, unless serving web pages.
	CorsEnabled bool `yaml:"corsEnabled"`

	// When CORS is enabled, allow these domain names. (eg https://*.otherdomain.com)
	CorsAllowedOrigins []string `yaml:"corsAllowedOrigins"`

	// Enable gzip compression
	GZipEnabled bool `yaml:"gzipEnabled,omitempty"`

	// GZip compression level when enabled -1..9
	GZipLevel int `yaml:"gzipLevel"`

	// GZip compression content types
	GZipContentTypes []string `yaml:"gzipContentTypes,omitempty"`

	// ServeFilesDir for use as file server or "" to not serve files (default)
	// This must be a full directory path.
	// ServeFilesDir string `yaml:"serveFilesDir"`
	// ServeFilesEndpoint endpoint of the file server.
	// Defaults to /static
	// ServeFilesEndpoint string `yaml:"serveFilesEndpoint"`

	// Customization handlers for logging, recovery and authentication

	// Optional middleware logger
	// Defaults to chi middelware.Logger
	// alternative: https://github.com/goware/httplog
	Logger func(http.Handler) http.Handler `yaml:"-"`

	// Recover from panics and return 500 error
	// Defaults to chi middleware.Recoverer
	// Set to nil to disable recovery and crash on panic.
	Recoverer func(http.Handler) http.Handler `yaml:"-"`

	// DisableStStripSlashesEnabledripSlashes remove trailing '/' in path
	StripSlashesEnabled bool `yaml:"stripSlashesEnabled,omitempty"`

	// Bearer token authenticator for protected routes
	// Defaults to blocking all requests.
	// Set to nil to add your own authn to the protected route.
	ValidateToken func(token string) (
		clientID string, sessionID string, err error) `yaml:"-"`
}

// create options with defaults
func NewHttpServerConfig() *HttpServerConfig {
	o := &HttpServerConfig{
		Address: "",
		Port:    8444,
		//
		CorsEnabled:        false,
		CorsAllowedOrigins: []string{"*"}, // replace this when enabling cors

		// Gzip compression is enabled by default
		GZipEnabled: true,
		// compression level br:0..11, gzip: -1..9
		GZipLevel: 5, // -1..9
		// That http response must have its content-type set for this to work
		GZipContentTypes: []string{
			"text/html",
			"text/css",
			"text/plain",
			"text/javascript",
			"application/javascript",
			"application/x-javascript",
			"application/json",
			"image/svg+xml"},
		Logger:              middleware.Logger,
		Authenticate:        nil,
		NoTLS:               false,
		Recoverer:           middleware.Recoverer,
		StripSlashesEnabled: true,
	}
	return o
}

// IHttpServer is the HTTP TLS server transport interface
type IHttpServer interface {
	modules.IHiveModule

	// Returns the connection URL of the http server
	GetConnectURL() string

	// Return the authenticated client from the http request context
	// The clientID is set in the middleware chain which includes auth check
	GetClientIdFromContext(r *http.Request) (string, error)

	// GetRequestParams decode the HiveOT standardized request parameters:
	// - clientID from context, provided by ?
	// - connectionID from the 'cid' header
	// - correlationID from the 'correlationID' header
	// - payload from the message body
	// - thingID, operation, name from URI variables
	GetRequestParams(r *http.Request) (RequestParams, error)

	// Return the protected route for adding endpoints.
	// This requires that config.Authenticate method is set, otherwise this path is not protected.
	GetProtectedRoute() chi.Router

	// Return the public route for adding endpoints.
	GetPublicRoute() chi.Router
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

	// Patch is the http convenience function to partially update a resource
	Patch(path string, body []byte) (output []byte, statusCode int, err error)

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
}

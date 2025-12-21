package httpserver

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

const DefaultHttpServerModuleID = "httpserver"

// The default listening port if none is set
const DefaultPort = 8444

// The context ID's for authenticated clientID and sessionID
const ClientContextID = "clientID"
const SessionContextID = "sessionID"

// HTTP TLS server transport interface
type IHttpsServer interface {
	// Return the router used by the TLS server.
	// Intended to let services add their endpoints.
	//
	// Local use only. nil when queried remotely.
	GetProtectedRouter() *chi.Mux
	// Return the protected
	GetPublicRouter() *chi.Mux
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

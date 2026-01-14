package httptransport

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/utils"
)

// Configuration options for the https server
type HttpServerConfig struct {
	Address    string            `yaml:"address,omitempty"`
	Port       int               `yaml:"port,omitempty"`
	CaCert     *x509.Certificate `yaml:"-"`
	ServerCert *tls.Certificate  `yaml:"-"`

	// NoTLS disables the use of TLS. For testing obviously
	NoTLS bool `yaml:"noTLS,omitempty"`

	// AuthenticateHandler authenticate requests on the protected route.
	//
	// This is optional for using a custom authentication mechanism.
	// This defaults to an internal function that takes the bearer token
	// from the request and passes it to the ValidateToken from this configuration.
	//
	// Note that ValidateToken is required when using the default handler.
	//
	// Other authentication schemes can be implemented by providing your own
	// function here.
	AuthenticateHandler func(req *http.Request) (clientID string, sessionID string, err error) `yaml:"-"`

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

	// Bearer token authenticator for protected routes.
	// This defaults to blocking all requests.
	//
	// Set to a custom function to perform actual token authentication.
	// any transports.IAuthenticator implementation can provide a ValidateToken function.
	ValidateToken transports.ValidateTokenHandler
}

// create options with defaults
//
//	addr is optional address, default is outbound address
//	port is optional listening port, 0 for default 8444
//	serverCert TLS certificate signed by the CA
//	caCert x509 CA certificate
//	validateToken is the required handler for authenticating protected routes
func NewHttpServerConfig(
	addr string, port int, serverCert *tls.Certificate, caCert *x509.Certificate,
	validateToken transports.ValidateTokenHandler) *HttpServerConfig {

	if addr == "" {
		addr = utils.GetOutboundIP("").String()
	}
	if port == 0 {
		port = 8444
	}

	o := &HttpServerConfig{
		Address:    addr,
		Port:       port,
		ServerCert: serverCert,
		CaCert:     caCert,
		//
		AuthenticateHandler: nil, // use default handler provided by server
		CorsEnabled:         false,
		CorsAllowedOrigins:  []string{"*"}, // replace this when enabling cors

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
		NoTLS:               false,
		Recoverer:           middleware.Recoverer,
		StripSlashesEnabled: true,
		ValidateToken:       validateToken,
	}
	return o
}

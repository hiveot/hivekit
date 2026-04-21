package transports

import (
	"context"
	"crypto/tls"
	"net/http"
	"time"

	"golang.org/x/net/http2"
)

// The default wait timeout for connecting. Use SetTimeout() to override.
const DefaultClientTimeout = time.Second * 60

// ITLSClient interface for generic http/tls client.
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
type ITLSClient interface {

	// Close the connection and release resources
	Close()

	// Connect using a client certificate
	// This does not make any calls yet, just sets the client certificate.
	// This returns an error if no CA is set
	ConnectWithClientCert(clientCert *tls.Certificate) (err error)

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

// Package tlsclient with a TLS client helper supporting certificate, JWT or Basic authentication
package tlsclient

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"

	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/teris-io/shortid"
	"golang.org/x/net/http2"
	"golang.org/x/net/publicsuffix"
)

// The default wait timeout for connecting. Use SetTimeout() to override.
const DefaultClientTimeout = time.Second * 60

// TLSClient is a simple TLS Client with authentication using certificates or JWT authentication with login/pw
type TLSClient struct {

	// Authorization header bearer token
	bearerToken string

	// The CA certificate to verify the server connected
	caCert *x509.Certificate

	// client certificate mutual authentication
	clientCert *tls.Certificate

	// The client this identifies as
	clientID string

	// connectionID
	cid string

	// optional customHeaders to include in each request
	customHeaders map[string]string

	// host:port of the server to setup to
	hostPort string

	// the native http client
	httpClient *http.Client
	// http2 transport
	tlsTransport *http2.Transport

	timeout time.Duration
}

// Close the connection with the server
func (cl *TLSClient) Close() {
	slog.Debug("TLSClient.Remove: Closing client connection")

	if cl.httpClient != nil {
		cl.httpClient.CloseIdleConnections()
		//cl.httpClient = nil
	}
}

// ConnectWithClientCert creates a connection with the server using a client certificate for mutual authentication.
// The provided certificate must be signed by the server's CA.
//
//	kp is the key-pair used to the certificate validation
//	clientCert client tls certificate containing x509 cert and private key
//
// Returns nil if successful, or an error if connection failed
//
//	func (cl *TLSClient) ConnectWithClientCert(kp keys.IHiveKey, clientCert *tls.Certificate) (err error) {
//		cl.mux.RLock()
//		defer cl.mux.RUnlock()
//		_ = kp
//		cl.tlsClient = tlsclient.NewTLSClient(cl.hostPort, clientCert, cl.caCert, cl.timeout)
//		return err
//	}

// Connect the client to a server with the given clientID and token.
//
// This does not yet make any http calls, just sets the parameters used for
// Send requests.
//
// This creates a unique connectionID for the header and places the token in
// the authorization hedaer.
func (cl *TLSClient) ConnectWithToken(clientID string, token string) error {
	// ensure disconnected (note that this resets retryOnDisconnect)
	cl.bearerToken = token
	cl.clientID = clientID
	cl.cid = shortid.MustGenerate()
	return nil
}

// Create a new http request with all the headers including authorization.
// The request can be cancelled with the given cancel function.
//
//	ctx optional context or nil for background.
//	method is the http GET/POST/... for request
//	path is the request path
//	qParams are optional query parameters
//	body is the optional request payload
//	contentType of the payload, default is application/json
//
// This returns the request, ready to be submitted, a cancel function or an error
func (cl *TLSClient) CreateRequest(
	ctx context.Context,
	method string, path string, qParams map[string]string,
	body []byte, contentType string,
) (req *http.Request) {

	// Step 1: create a request object
	if contentType == "" {
		contentType = "application/json"
	}
	if ctx == nil {
		ctx = context.Background()
	}
	// Caution! a double // in the path causes a 301 and changes post to get
	fullURL := fmt.Sprintf("https://%s%s", cl.hostPort, path)

	bodyReader := bytes.NewReader(body)
	r, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		slog.Error("Send, bad request method or URL", "method", method, "fullURL", fullURL)
		return nil
	}

	// Step 2: add headers for origin, authorization, content-Type
	origin := "https://" + cl.hostPort
	r.Header.Set("Origin", origin)

	r.Header.Set("Content-Type", contentType)

	if cl.bearerToken != "" {
		r.Header.Add("Authorization", "bearer "+cl.bearerToken)
	}
	if cl.cid != "" {
		r.Header.Add(transports.ConnectionIDHeader, cl.cid)
	}

	//  any custom headers
	for k, v := range cl.customHeaders {
		r.Header.Set(k, v)
	}

	// Step 3: optional query parameters
	if qParams != nil {
		qValues := r.URL.Query()
		for k, v := range qParams {
			qValues.Add(k, v)
		}
		r.URL.RawQuery = qValues.Encode()
	}
	return r
}

// Delete sends a delete message
// Note that delete methods do not allow a body, or a 405 is returned.
//
//	path to invoke
func (cl *TLSClient) Delete(path string) (httpStatus int, err error) {
	// careful, a double // in the path causes a 301 and changes POST to GET
	ctx, cancelFn := context.WithTimeout(context.Background(), cl.timeout)
	_, httpStatus, _, err = cl.Send(ctx, "DELETE", path, nil, nil, "")
	cancelFn()
	return httpStatus, err

}

// Get is a convenience function to read a resource.
// This returns the response data, the http status code and an error of delivery failed
//
//	path to invoke
func (cl *TLSClient) Get(path string) (resp []byte, httpStatus int, err error) {
	ctx, cancelFn := context.WithTimeout(context.Background(), cl.timeout)
	resp, httpStatus, _, err = cl.Send(ctx, "GET", path, nil, nil, "")
	cancelFn()
	return resp, httpStatus, err
}

func (cl *TLSClient) GetClientCertificate() *tls.Certificate {
	return cl.clientCert
}

func (cl *TLSClient) GetClientID() string {
	return cl.clientID
}
func (cl *TLSClient) GetConnectionID() string {
	return cl.cid
}
func (cl *TLSClient) GetHostPort() string {
	return cl.hostPort
}

// GetHttpClient returns the native HTTP client
func (cl *TLSClient) GetHttpClient() *http.Client {
	return cl.httpClient
}

// GetHttpClient returns the native HTTP client
func (cl *TLSClient) GetTlsTransport() *http2.Transport {
	return cl.tlsTransport
}

// HttpConnect - send a http connect request (for proxies)
func (cl *TLSClient) HttpConnect() (statusCode int, err error) {

	ctx, cancelFn := context.WithTimeout(context.Background(), cl.timeout)
	_, statusCode, _, err = cl.Send(ctx, http.MethodConnect, "", nil, nil, "")
	cancelFn()
	return statusCode, err
}

// HttpConnect
func (cl *TLSClient) Head(path string) (statusCode int, err error) {

	ctx, cancelFn := context.WithTimeout(context.Background(), cl.timeout)
	_, statusCode, _, err = cl.Send(ctx, http.MethodHead, path, nil, nil, "")
	cancelFn()
	return statusCode, err
}

//// Logout from the server and end the session
//func (cl *TLSClient) Logout() error {
//	serverURL := fmt.Sprintf("https://%s%s", cl.hostPort, vocab.PostLogoutPath)
//	_, err := cl._send("POST", serverURL, http.NoBody, nil)
//	return err
//}

// Ping sends a ping request to the server on the well-known /ping endpoint
func (cl *TLSClient) Ping() (statusCode int, err error) {

	ctx, cancelFn := context.WithTimeout(context.Background(), cl.timeout)
	_, statusCode, _, err = cl.Send(ctx, http.MethodGet, transports.DefaultPingPath, nil, nil, "")
	cancelFn()
	return statusCode, err
}

// Patch sends a patch message with json payload
// If msg is a string then it is considered to be already serialized.
// If msg is not a string then it will be json encoded.
//
//	path to invoke
//	body contains the serialized body
func (cl *TLSClient) Patch(
	path string, body []byte) (resp []byte, statusCode int, err error) {

	// careful, a double // in the path causes a 301 and changes POST to GET
	ctx, cancelFn := context.WithTimeout(context.Background(), cl.timeout)
	resp, statusCode, _, err = cl.Send(ctx, http.MethodPatch, path, nil, body, "")
	cancelFn()
	return resp, statusCode, err
}

// Post a message.
// If msg is a string then it is considered to be already serialized.
// If msg is not a string then it will be json encoded.
//
//	path to invoke
//	body contains the serialized request body
//
// This returns the serialized response data
func (cl *TLSClient) Post(path string, body []byte) (
	resp []byte, statusCode int, err error) {

	ctx, cancelFn := context.WithTimeout(context.Background(), cl.timeout)
	resp, statusCode, _, err = cl.Send(ctx, http.MethodPost, path, nil, body, "")
	cancelFn()
	return resp, statusCode, err
}

// PostForm posts a form message.
func (cl *TLSClient) PostForm(path string, formData map[string]string) (
	resp []byte, statusCode int, err error) {

	form := url.Values{}
	for k, v := range formData {
		form.Add(k, v)
	}
	body := form.Encode()
	ctx, cancelFn := context.WithTimeout(context.Background(), cl.timeout)
	resp, statusCode, _, err = cl.Send(ctx, http.MethodPost, path, nil,
		[]byte(body), "application/x-www-form-urlencoded")
	cancelFn()
	return resp, statusCode, err
}

// Put a message with json payload
// If msg is a string then it is considered to be already serialized.
// If msg is not a string then it will be json encoded.
//
//	path to invoke
//	body contains the serialized request body
//	correlationID optional field to link async requests and responses
func (cl *TLSClient) Put(path string, body []byte) (
	resp []byte, statusCode int, err error) {

	// careful, a double // in the path causes a 301 and changes POST to GET
	ctx, cancelFn := context.WithTimeout(context.Background(), cl.timeout)
	resp, statusCode, _, err = cl.Send(ctx, http.MethodPut, path, nil, body, "")
	cancelFn()
	return resp, statusCode, err
}

// Send a HTTPS request and read response.
//
// If a JWT authentication is enabled then add the bearer token to the header
// If msg is a string then it is considered to be already serialized.
// If msg is not a string then it will be json encoded.
//
//	method: GET, PUT, POST, ...
//	path: path of URL
//	body contains the serialized request body
//	contentType: default is "application/json"
//	qParams: optional map with query parameters
//
// This returns the serialized response data, a response message ID, return status code or an error
func (cl *TLSClient) Send(
	ctx context.Context,
	method string, path string, qParams map[string]string, body []byte, contentType string) (
	resp []byte, httpStatus int, headers http.Header, err error) {

	if cl == nil || cl.httpClient == nil {
		err = fmt.Errorf("send: %s %s. Client is not started", method, path)
		return nil, http.StatusInternalServerError, nil, err
	}
	// ctx, cancelFn := context.WithTimeout(context.Background(), cl.timeout)
	httpRequest := cl.CreateRequest(ctx, method, path, qParams, body, contentType)
	// _ = cancelFn

	httpResp, err := cl.httpClient.Do(httpRequest)
	if err != nil {
		err = fmt.Errorf("Send: %s %s: %w", method, path, err)
		slog.Error(err.Error())
		return nil, 500, nil, err
	} else if httpResp.StatusCode >= 300 {
		err = fmt.Errorf("Send: %s %s: failed with (%d) %s",
			method, path, httpResp.StatusCode, httpResp.Status)
		slog.Error(err.Error())
		return nil, httpResp.StatusCode, nil, err
	}
	respBody, err := io.ReadAll(httpResp.Body)
	// response body MUST be closed
	_ = httpResp.Body.Close()
	httpStatus = httpResp.StatusCode

	if httpStatus == 401 {
		err = fmt.Errorf("%s", httpResp.Status)
	} else if httpStatus >= 400 && httpStatus < 500 {
		err = fmt.Errorf("%s: %s", httpResp.Status, respBody)
		if httpResp.Status == "" {
			err = fmt.Errorf("%d (%s): %s", httpResp.StatusCode, httpResp.Status, respBody)
		}
	} else if httpStatus >= 500 {
		err = fmt.Errorf("Error %d (%s): %s", httpStatus, httpResp.Status, respBody)
		slog.Error("Send returned internal server error",
			"url", httpRequest.RemoteAddr, "err", err.Error())
	} else if err != nil {
		err = fmt.Errorf("Send: Error %s %s: %w", method, httpRequest.URL.String(), err)
	}
	return respBody, httpStatus, httpResp.Header, err
}

// SetHeader sets a custom header to include in each request
// use an empty value to remove the header
func (cl *TLSClient) SetHeader(name string, val string) {
	if val == "" {
		delete(cl.customHeaders, name)
	} else {
		cl.customHeaders[name] = val
	}
}

// SetTimeout overrides the default timeout for connecting and sending messages
func (cl *TLSClient) SetTimeout(timeout time.Duration) {
	cl.timeout = timeout
}

// Trace performs a message loopback of the target resource
func (cl *TLSClient) Trace(path string) (statusCode int, err error) {

	ctx, cancelFn := context.WithTimeout(context.Background(), cl.timeout)
	_, statusCode, _, err = cl.Send(ctx, http.MethodTrace, path, nil, nil, "")
	cancelFn()
	return statusCode, err
}

// NewTLSClient creates a new TLS Client instance.
// Use setup/Remove to open and close connections
//
//	hostPort is the server address in host:port format
//	clientCert is an optional client certificate used to authenticate. cert Subject is used as clientID
//	caCert with the x509 CA certificate, nil if not available
//	timeout duration for use with Delete,Get,Patch,Post,Put, 0 for DefaultClientTimeout
//
// returns TLS client for submitting requests
func NewTLSClient(hostPort string,
	clientCert *tls.Certificate, caCert *x509.Certificate, timeout time.Duration) *TLSClient {

	var clientID string
	if timeout == 0 {
		timeout = DefaultClientTimeout
	}
	// Use CA certificate for server authentication if it exists
	if caCert == nil {
		slog.Info("NewTLSClient: No CA certificate. InsecureSkipVerify used",
			slog.String("destination", hostPort))
	}

	var clientCertList []tls.Certificate
	caCertPool := x509.NewCertPool()
	if caCert != nil {
		caCertPool.AddCert(caCert)
	}
	if clientCert != nil {
		clientCertList = []tls.Certificate{*clientCert}

		//--- verify the client certificate against the CA
		// if a client cert is given then test if it is valid for our CA.
		// this detects problems with certs that can be hard to track down
		opts := x509.VerifyOptions{
			Roots:     caCertPool,
			KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		}
		x509Cert, err := x509.ParseCertificate(clientCert.Certificate[0])
		if err == nil {
			// FIXME: TestCertAuth: certificate specifies incompatible key usage
			// why? Is the certpool invalid? Yet the test succeeds
			_, err = x509Cert.Verify(opts)
			// cert subject is clientID
			clientID = x509Cert.Subject.String()
		}
		if err != nil {
			err = fmt.Errorf("NewTLSClient: certificate verfication failed: %w. Continuing for now.", err)
			slog.Error(err.Error())
		}
		//--- end verify
	}
	tlsConfig := &tls.Config{
		RootCAs: caCertPool,
		// why is ServerName not required?
		InsecureSkipVerify: caCert == nil,
		Certificates:       clientCertList,
	}

	// create the http/2 transport
	tlsTransport := &http2.Transport{
		AllowHTTP: true, // false to disable http/2 over cleartext TCP
		DialTLSContext: func(ctx context.Context, network, addr string, cfg *tls.Config) (net.Conn, error) {
			c, err := tls.Dial(network, addr, cfg)
			return c, err
		},
		TLSClientConfig: tlsConfig,
	}
	// var cn tls.Conn = tlsTransport
	// _ = cn

	// add a cookie jar for storing cookies
	cjarOpts := &cookiejar.Options{PublicSuffixList: publicsuffix.List}
	cjar, err := cookiejar.New(cjarOpts)
	if err != nil {
		err = fmt.Errorf("NewHttp2TLSClient: error creating cookiejar. Continuing anyways: %w", err)
		slog.Error(err.Error())
		err = nil
	}
	// Dont set a timeout here as it will end the connection
	httpClient := &http.Client{
		Transport: tlsTransport,
		Jar:       cjar,
	}

	cl := &TLSClient{
		clientID:      clientID, // only set through client certificate
		hostPort:      hostPort,
		httpClient:    httpClient,
		timeout:       timeout,
		clientCert:    clientCert,
		caCert:        caCert,
		customHeaders: make(map[string]string),
	}

	// interface check
	var _ transports.ITlsClient = cl

	return cl
}

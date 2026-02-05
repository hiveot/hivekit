package tlsclient_test

import (
	"crypto"
	"crypto/tls"
	"crypto/x509"
	"log/slog"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/hiveot/hivekit/go/lib/logging"
	"github.com/hiveot/hivekit/go/modules/certs/module/selfsigned"
	tlsclient "github.com/hiveot/hivekit/go/modules/transports/httpserver/client"
	"github.com/hiveot/hivekit/go/utils"
	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

// test hostname and port
var testAddress string
var TestKeyType = utils.KeyTypeED25519

// CA, server and plugin test certificate
var authBundle selfsigned.TestCertBundle
var serverTLSConf *tls.Config

// x509CertToTLS combines a x509 certificate and private key into a TLS certificate
func x509CertToTLS(cert *x509.Certificate, privKey crypto.PrivateKey) *tls.Certificate {
	// A TLS certificate is a wrapper around x509 with private key
	tlsCert := tls.Certificate{}
	tlsCert.Certificate = append(tlsCert.Certificate, cert.Raw)
	tlsCert.PrivateKey = privKey

	return &tlsCert
}

func startTestServer(mux *http.ServeMux) (*http.Server, error) {
	var err error
	httpServer := &http.Server{
		Addr: testAddress,
		// ReadTimeout:  5 * time.Minute, // 5 min to allow for delays when testing
		// WriteTimeout: 10 * time.Second,
		// Handler:   srv.router,
		TLSConfig: serverTLSConf,
		Handler:   mux,
		//ErrorLog:  log.Default(),
	}
	go func() {
		err = httpServer.ListenAndServeTLS("", "")
	}()
	// Catch any startup errors
	time.Sleep(100 * time.Millisecond)
	return httpServer, err
}

// TestMain runs a http server
// Used for all test cases in this package
func TestMain(m *testing.M) {
	logging.SetLogging("info", "")
	slog.Info("------ TestMain of httpauthhandler ------")
	testAddress = "127.0.0.1:9888"
	// hostnames := []string{testAddress}

	authBundle = selfsigned.CreateTestCertBundle(TestKeyType)

	caCertPool := x509.NewCertPool()
	caCertPool.AddCert(authBundle.CaCert)

	// serverTLSCert := testenv.X509ToTLS(certsclient.ServerCert, nil)
	serverTLSConf = &tls.Config{
		Certificates:       []tls.Certificate{*authBundle.ServerCert},
		ClientAuth:         tls.VerifyClientCertIfGiven,
		ClientCAs:          caCertPool,
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: false,
	}

	res := m.Run()

	time.Sleep(time.Second)
	os.Exit(res)
}

func TestNoCA(t *testing.T) {
	path1 := "/hello"
	path1Hit := 0

	// setup server and client environment
	mux := http.NewServeMux()
	srv, err := startTestServer(mux)
	mux.HandleFunc(path1, func(http.ResponseWriter, *http.Request) {
		slog.Info("TestAuthCert: path1 hit")
		path1Hit++
	})
	assert.NoError(t, err)

	// certificate authentication but no CA
	cl := tlsclient.NewTLSClient(testAddress, authBundle.ClientCert, nil, 0)
	assert.NoError(t, err)

	_, _, err = cl.Get(path1)
	assert.NoError(t, err)
	assert.Equal(t, 1, path1Hit)
	cl.Close()

	// No authentication
	cl = tlsclient.NewTLSClient(testAddress, nil, nil, 0)

	_, _, err = cl.Get(path1)
	assert.NoError(t, err)
	assert.Equal(t, 2, path1Hit)

	cl.Close()
	_ = srv.Close()
}

// Test certificate based authentication
func TestAuthClientCert(t *testing.T) {
	path1 := "/test1"
	path1Hit := 0

	// setup server and client environment
	mux := http.NewServeMux()
	srv, err := startTestServer(mux)
	assert.NoError(t, err)
	//
	mux.HandleFunc(path1, func(http.ResponseWriter, *http.Request) {
		slog.Info("TestAuthClientCert: path1 hit")
		path1Hit++
	})
	//
	cl := tlsclient.NewTLSClient(testAddress, authBundle.ClientCert, authBundle.CaCert, 0)
	assert.NoError(t, err)

	clientCert := cl.GetClientCertificate()
	assert.NotNil(t, clientCert)

	// verify service certificate against CA
	caCertPool := x509.NewCertPool()
	caCertPool.AddCert(authBundle.CaCert)
	opts := x509.VerifyOptions{
		Roots:     caCertPool,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	cert, err := x509.ParseCertificate(clientCert.Certificate[0])
	if err == nil {
		_, err = cert.Verify(opts)
	}
	assert.NoError(t, err)

	//
	_, _, err = cl.Get(path1)
	assert.NoError(t, err)
	_, _, err = cl.Post(path1, nil)
	assert.NoError(t, err)
	_, _, err = cl.Put(path1, nil)
	assert.NoError(t, err)
	_, err = cl.Delete(path1)
	assert.NoError(t, err)
	_, _, err = cl.Patch(path1, nil)
	assert.NoError(t, err)
	assert.Equal(t, 5, path1Hit)

	cl.Close()
	_ = srv.Close()
}

func TestNotStarted(t *testing.T) {
	cl := tlsclient.NewTLSClient(testAddress, nil, authBundle.CaCert, 0)
	_, _, err := cl.Get("/notstarted")
	assert.Error(t, err)
	cl.Close()
}
func TestNoClientCert(t *testing.T) {
	cl := tlsclient.NewTLSClient(testAddress, nil, authBundle.CaCert, 0)
	cl.Close()
}

func TestBadClientCert(t *testing.T) {
	// use cert not signed by the CA
	otherCA, otherPrivKey, _, err := selfsigned.CreateSelfSignedCA(
		"", "", "", "", "", 1, TestKeyType)
	otherCert, err := selfsigned.CreateClientCert("name", "ou", 1,
		authBundle.ClientPubKey, otherCA, otherPrivKey)
	require.NoError(t, err)
	otherTLS := x509CertToTLS(otherCert, authBundle.ClientPrivKey)

	cl := tlsclient.NewTLSClient(testAddress, otherTLS, authBundle.CaCert, 0)
	// this should produce an error in the log
	//assert.Error(t, err)
	cl.Close()
}

func TestNoServer(t *testing.T) {
	// setup server and client environm
	//
	cl := tlsclient.NewTLSClient(testAddress, authBundle.ClientCert, authBundle.CaCert, 0)
	_, _, err := cl.Get("/noserver")
	assert.Error(t, err)
	cl.Close()
}
func TestCert404(t *testing.T) {
	mux := http.NewServeMux()
	srv, err := startTestServer(mux)
	assert.NoError(t, err)

	cl := tlsclient.NewTLSClient(testAddress, authBundle.ClientCert, authBundle.CaCert, 0)

	_, _, err = cl.Get("/pathnotfound")
	assert.Error(t, err)

	cl.Close()
	_ = srv.Close()
}

func TestTokenAuth(t *testing.T) {
	pathLogin1 := "/login"
	user1 := "user1"
	authToken := "some-auth-token"
	testResponse := "some-response"
	var testResult string

	// setup server and client environment
	mux := http.NewServeMux()

	// Handle a jwt login
	mux.HandleFunc(pathLogin1, func(w http.ResponseWriter, req *http.Request) {

		// expect a valid bearer token
		rxToken, err := utils.GetBearerToken(req)
		require.NoError(t, err)
		assert.Equal(t, authToken, rxToken)
		utils.WriteReply(w, true, testResponse, err)
	})

	srv, err := startTestServer(mux)
	assert.NoError(t, err)

	// connect using the given token
	cl := tlsclient.NewTLSClient(testAddress, authBundle.ClientCert, authBundle.CaCert, 0)
	err = cl.ConnectWithToken(user1, authToken)
	require.NoError(t, err)

	resp, status, err := cl.Post(pathLogin1, nil)
	require.NoError(t, err)
	assert.Equal(t, status, http.StatusOK)
	err = jsoniter.Unmarshal(resp, &testResult)
	require.NoError(t, err)
	assert.Equal(t, testResponse, testResult)

	cl.Close()
	_ = srv.Close()
}

func TestTokenFail(t *testing.T) {
	pathHello1 := "/hello"
	clientID := "user1"

	// setup server and client environment
	mux := http.NewServeMux()
	srv, err := startTestServer(mux)
	assert.NoError(t, err)
	//
	mux.HandleFunc(pathHello1, func(resp http.ResponseWriter, req *http.Request) {
		slog.Info("TestTokenFail: login")
		//_, _ = resp.Write([]byte("invalid token"))
		resp.WriteHeader(http.StatusUnauthorized)
	})
	//
	cl := tlsclient.NewTLSClient(testAddress, nil, authBundle.CaCert, 0)
	cl.ConnectWithToken(clientID, "badtoken")
	resp, _, err := cl.Post(pathHello1, []byte("test"))
	assert.Empty(t, resp)
	// unauthorized
	assert.Error(t, err)

	cl.Close()
	_ = srv.Close()
}

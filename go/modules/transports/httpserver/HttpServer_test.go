package httpserver_test

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/hiveot/hivekit/go/modules/certs/module/selfsigned"
	"github.com/hiveot/hivekit/go/modules/transports/httpserver"
	tlsclient "github.com/hiveot/hivekit/go/modules/transports/httpserver/client"
	"github.com/hiveot/hivekit/go/modules/transports/httpserver/module"
	"github.com/hiveot/hivekit/go/utils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var serverAddress string
var serverPort int = 9445
var clientHostPort string
var testCerts selfsigned.TestCertBundle
var TestKeyType = utils.KeyTypeED25519

// TestMain runs a http server
// Used for all test cases in this package
func TestMain(m *testing.M) {
	utils.SetLogging("info", "")
	// slog.Info("------ TestMain of TLSServer_test.go ------")
	// serverAddress = utils.GetOutboundIP("").String()
	// use the localhost interface for testing
	serverAddress = "127.0.0.1"

	// hostnames := []string{serverAddress}
	clientHostPort = fmt.Sprintf("%s:%d", serverAddress, serverPort)

	testCerts = selfsigned.CreateTestCertBundle(TestKeyType)
	res := m.Run()

	time.Sleep(time.Second)
	os.Exit(res)
}

func TestStartStop(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	cfg := httpserver.NewHttpServerConfig(
		serverAddress, serverPort, testCerts.ServerCert, testCerts.CaCert, nil)
	srv := module.NewHttpServerModule("", cfg)
	err := srv.Start()
	assert.NoError(t, err)
	srv.Stop()
}

func TestNoServerCert(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	cfg := httpserver.NewHttpServerConfig(
		serverAddress, serverPort, nil, testCerts.CaCert, nil)

	srv := module.NewHttpServerModule("", cfg)
	err := srv.Start()
	require.Error(t, err)
	srv.Stop()
}

// connect without authentication
func TestNoAuth(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	path1 := "/hello"
	path1Hit := 0

	cfg := httpserver.NewHttpServerConfig(
		serverAddress, serverPort, testCerts.ServerCert, testCerts.CaCert, nil)

	srv := module.NewHttpServerModule("", cfg)

	err := srv.Start()
	require.NoError(t, err)
	defer srv.Stop()

	router := srv.GetPublicRoute()
	router.Get(path1, func(w http.ResponseWriter, req *http.Request) {
		// expect no bearer token
		bearerToken, err := utils.GetBearerToken(req)
		assert.Error(t, err)
		assert.Empty(t, bearerToken)
		slog.Info("TestNoAuth: path1 hit")
		path1Hit++
	})

	cl := tlsclient.NewTLSClient(clientHostPort, nil, testCerts.CaCert, 0)
	_, _, err = cl.Get(path1)
	assert.NoError(t, err)
	assert.Equal(t, 1, path1Hit)

	cl.Close()
}

func TestTokenAuth(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	path1 := "/test1"
	path1Hit := 0
	loginID1 := "user1"
	token1 := "abcd"
	badToken := "badtoken"
	validUntil := time.Now()

	// setup server and client environment
	cfg := httpserver.NewHttpServerConfig(
		serverAddress, serverPort, testCerts.ServerCert, testCerts.CaCert, nil)

	cfg.ValidateTokenHandler = func(bearerToken string) (string, string, time.Time, error) {
		assert.NotEmpty(t, bearerToken)
		clientID := "bob"
		return clientID, "", validUntil, nil //fmt.Errorf("test fail")
	}
	srv := module.NewHttpServerModule("", cfg)
	err := srv.Start()
	require.NoError(t, err)
	defer srv.Stop()

	//srv.EnableBasicAuth(func(userID, password string) bool {
	//	path1Hit++
	//	return userID == loginID1 && password == password1
	//})
	// router := srv.GetPublicRouter()
	router := srv.GetProtectedRoute()
	router.Get(path1, func(w http.ResponseWriter, req *http.Request) {
		// expect a bearer token
		bearerToken, err := utils.GetBearerToken(req)
		assert.NoError(t, err)
		if bearerToken == token1 {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusUnauthorized)
		}
		slog.Info("TestBearerAuth: path1 hit")
		path1Hit++
	})

	// create a client and login
	cl := tlsclient.NewTLSClient(clientHostPort, nil, testCerts.CaCert, 0)
	require.NoError(t, err)
	defer cl.Close()
	cl.ConnectWithToken(loginID1, token1)

	// test the auth with a GET request
	_, _, err = cl.Get(path1)
	assert.NoError(t, err)
	assert.Equal(t, 1, path1Hit)

	// test a failed login
	cl.Close()
	cl.ConnectWithToken(loginID1, badToken)
	_, _, err = cl.Get(path1)
	assert.Error(t, err)
	assert.Equal(t, 2, path1Hit) // should not increase

}

func TestClientCert(t *testing.T) {
	path1 := "/hello"
	path1Hit := 0
	// srv, router := service.NewTLSServer(serverAddress, serverPort,
	// 	testCerts.ServerCert, testCerts.CaCert)

	cfg := httpserver.NewHttpServerConfig(
		serverAddress, serverPort, testCerts.ServerCert, testCerts.CaCert, nil)
	srv := module.NewHttpServerModule("", cfg)

	err := srv.Start()
	assert.NoError(t, err)
	// handler can be added any time
	routes := srv.GetPublicRoute()
	routes.Get(path1, func(w http.ResponseWriter, r *http.Request) {
		slog.Info("TestAuthCert: path1 hit")

		// test getting client cert
		clcerts := r.TLS.PeerCertificates
		if len(clcerts) == 0 {
			assert.Fail(t, "missing client cert")
		}
		clientCert := clcerts[0]
		clientID := clientCert.Subject.CommonName
		assert.Equal(t, testCerts.ClientID, clientID)

		params, err := module.GetRequestParams(r)
		assert.Equal(t, testCerts.ClientID, params.ClientID)
		require.NoError(t, err)
		require.NotNil(t, params)

		path1Hit++
	})

	cl := tlsclient.NewTLSClient(clientHostPort, testCerts.ClientCert, testCerts.CaCert, 0)
	_, status, err := cl.Get(path1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, 1, path1Hit)

	cl.Close()
	srv.Stop()
}

// Test valid authentication using JWT
//func TestQueryParams(t *testing.T) {
//	path2 := "/hello"
//	path2Hit := 0
//	srv, router := service.NewTLSServer(serverAddress, serverPort,
//		testCerts.ServerCert, testCerts.CaCert)
//	err := srv.Start()
//	assert.NoError(t, err)
//	srv.AddHandler(path2, func(userID string, resp http.ResponseWriter, req *http.Request) {
//		// query string
//		q1 := srv.GetQueryString(req, "query1", "")
//		assert.Equal(t, "bob", q1)
//		// fail not a number
//		_, err := srv.GetQueryInt(req, "query1", 0) // not a number
//		assert.Error(t, err)
//		// query of number
//		q2, _ := srv.GetQueryInt(req, "query2", 0)
//		assert.Equal(t, 3, q2)
//		// default should work
//		q3 := srv.GetQueryString(req, "query3", "default")
//		assert.Equal(t, "default", q3)
//		// multiple parameters fail
//		_, err = srv.GetQueryInt(req, "multi", 0)
//		assert.Error(t, err)
//		path2Hit++
//	})
//
//	cl := httpapi.NewTLSClient(clientHostPort, testCerts.CaCert)
//	require.NoError(t, err)
//	err = cl.ConnectWithClientCert(testCerts.ClientCert)
//	assert.NoError(t, err)
//
//	_, err = cl.Get(fmt.Sprintf("%s?query1=bob&query2=3&multi=a&multi=b", path2))
//	assert.NoError(t, err)
//	assert.Equal(t, 1, path2Hit)
//
//	cl.Remove()
//	srv.Stop()
//}

func TestWriteResponse(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	path2 := "/hello"
	message := "hello world"
	path2Hit := 0

	cfg := httpserver.NewHttpServerConfig(
		serverAddress, serverPort, testCerts.ServerCert, testCerts.CaCert, nil)
	srv := module.NewHttpServerModule("", cfg)

	err := srv.Start()
	assert.NoError(t, err)
	defer srv.Stop()
	router := srv.GetPublicRoute()

	router.Get(path2, func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte(message))
		w.WriteHeader(http.StatusOK)
		//srv.WriteBadRequest(resp, "bad request")
		//srv.WriteInternalError(resp, "internal error")
		//srv.WriteNotFound(resp, "not found")
		//srv.WriteNotImplemented(resp, "not implemented")
		//srv.WriteUnauthorized(resp, "unauthorized")
		path2Hit++
	})

	cl := tlsclient.NewTLSClient(clientHostPort, nil, testCerts.CaCert, 0)
	require.NoError(t, err)
	defer cl.Close()
	reply, _, err := cl.Get(path2)
	assert.NoError(t, err)
	assert.Equal(t, 1, path2Hit)
	assert.Equal(t, message, string(reply))
}

func TestBadPort(t *testing.T) {
	t.Logf("---%s---\n", t.Name())

	cfg := httpserver.NewHttpServerConfig(
		serverAddress, serverPort, testCerts.ServerCert, testCerts.CaCert, nil)

	cfg.Address = serverAddress
	cfg.Port = 1 // bad port
	cfg.CaCert = testCerts.CaCert
	cfg.ServerCert = testCerts.ServerCert
	srv := module.NewHttpServerModule("", cfg)

	err := srv.Start()
	defer srv.Stop()
	assert.Error(t, err)
}

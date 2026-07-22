package certstest_test

import (
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/hiveot/hivekit/go/modules/certs"
	certsclient "github.com/hiveot/hivekit/go/modules/certs/client"
	certsservice "github.com/hiveot/hivekit/go/modules/certs/service"
	certstest "github.com/hiveot/hivekit/go/modules/certs/test"
	"github.com/hiveot/hivekit/go/testenv"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var storageDir = filepath.Join(testenv.TestHome, "certs")

// private key type used in test
const TestKeyType = utils.KeyTypeECDSA

// start the certs service
func startService(t *testing.T) (certs.ICertsService, func(), error) {

	// clear start
	_ = os.RemoveAll(storageDir)
	cfg := certs.CertsConfig{CertsDir: storageDir}
	m := certsservice.NewCertsService(cfg)
	err := m.Start()
	require.NoError(t, err)
	return m, func() {
		m.Stop()
	}, err
}

// TestMain create a test folder for certificates and private key
func TestMain(m *testing.M) {

	utils.SetLogging("info", "")

	result := m.Run()
	if result != 0 {
		println("Test failed with code:", result)
		println("Find test files in:", storageDir)
	} else {
		// comment out the next line to be able to inspect results
		// _ = os.RemoveAll(storageDir)
	}

	os.Exit(result)
}

// Generic store store testcases
func TestStartStop(t *testing.T) {
	t.Logf("---%s---\n", t.Name())

	m, stopFn, err := startService(t)
	_ = m
	require.NoError(t, err)
	defer stopFn()
}

func TestX509ToFromPem(t *testing.T) {
	testCerts := certstest.CreateTestCertBundle(TestKeyType)
	asPem := utils.X509CertToPEM(testCerts.CaCert)
	assert.NotEmpty(t, asPem)
	asX509, err := utils.X509CertFromPEM(asPem)
	assert.NoError(t, err)
	assert.NotEmpty(t, asX509)
}

func TestSaveLoadX509Cert(t *testing.T) {
	// hostnames := []string{"localhost"}
	caPemFile := path.Join(storageDir, "caCert.pem")

	testCerts := certstest.CreateTestCertBundle(TestKeyType)

	// save the test x509 cert
	err := utils.SaveX509CertToPEM(testCerts.CaCert, caPemFile)
	assert.NoError(t, err)

	caCert, err := utils.LoadX509CertFromPEM(caPemFile)
	assert.NoError(t, err)
	assert.NotNil(t, caCert)

	// create a server TLS cert
	tlsCert := utils.X509CertToTLS(caCert, testCerts.CaPrivKey)
	assert.NotEmpty(t, tlsCert)

}

func TestPublicKeyFromCert(t *testing.T) {
	testCerts := certstest.CreateTestCertBundle(TestKeyType)
	kt, pubKey := utils.PublicKeyFromCert(testCerts.CaCert)
	assert.NotEmpty(t, pubKey)
	assert.NotEmpty(t, kt)
}

func TestSaveLoadTLSCert(t *testing.T) {
	// hostnames := []string{"localhost"}
	certFile := path.Join(storageDir, "x509cert.pem")
	keyFile := path.Join(storageDir, "tlskey.pem")

	testCerts := certstest.CreateTestCertBundle(TestKeyType)

	// save the test x509 part of the TLS cert
	err := utils.SaveTLSCertToPEM(testCerts.ServerCert, certFile, keyFile)
	assert.NoError(t, err)

	// load back the x509 part of the TLS cert
	cert, err := utils.LoadTLSCertFromPEM(certFile, keyFile)
	assert.NoError(t, err)
	assert.NotNil(t, cert)
}

func TestService(t *testing.T) {
	const appName = "server"

	m, cancelFn, err := startService(t)
	_ = m
	require.NoError(t, err)
	defer cancelFn()

	caCert, err := m.GetCACert()
	require.NoError(t, err)
	require.NotEmpty(t, caCert)

	// create the server cert and generate the keys
	newServerCert, err := m.CreateServerCert(appName, "", nil, nil)
	require.NoError(t, err)
	require.NotEmpty(t, newServerCert)

	tlsServerCert, err := m.LoadServerTLSCert(appName)
	require.NoError(t, err)
	require.NotEmpty(t, tlsServerCert)

	serverCert, serverKey, err := utils.TLSCertToX509(tlsServerCert)
	require.NoError(t, err)
	require.NotEmpty(t, serverKey)

	err = m.VerifyCert(appName, serverCert)
	require.NoError(t, err)
}

func TestMsgClient(t *testing.T) {
	m, cancelFn, err := startService(t)
	_ = m
	require.NoError(t, err)
	defer cancelFn()

	// use a direct transport instead of running a client-server
	tp := testenv.NewTestTransport("testclient", m)
	cl := certsclient.NewCertsMsgClient(tp, "")
	caCert, err := cl.GetCACert()
	require.NoError(t, err)
	require.NotEmpty(t, caCert)

	modCA, _ := m.GetCACert()
	err = m.VerifyCert(modCA.Subject.CommonName, caCert)
	assert.NoError(t, err)

	// assert.Equal(t, testCerts.CaCert.Issuer, caCert.Issuer.CommonName)
	// var _ certs.ICertsService = cl // interface check
	_ = cl
}

func TestCreateCerts(t *testing.T) {
	m, cancelFn, err := startService(t)
	_ = m
	require.NoError(t, err)
	defer cancelFn()

	caCert, err := m.GetCACert()
	require.NoError(t, err)
	require.NotNil(t, caCert)
	// require.NotNil(t, caPrivKey)

	serverTlsCert, err := m.CreateServerCert("test", "hostname", nil, nil)
	require.NoError(t, err)
	require.NotNil(t, serverTlsCert)

	// this needs completion
	cl := certsclient.NewCertsMsgClient(nil, "")
	// var _ certs.ICertsService = cl // interface check
	_ = cl
}

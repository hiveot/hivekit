package certs_test

import (
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/hiveot/hivekit/go/modules/certs"
	"github.com/hiveot/hivekit/go/modules/certs/certutils"
	certsclient "github.com/hiveot/hivekit/go/modules/certs/client"
	"github.com/hiveot/hivekit/go/modules/certs/module"
	"github.com/hiveot/hivekit/go/modules/certs/module/selfsigned"
	"github.com/hiveot/hivekit/go/modules/transports/direct"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var TestCertDir string

// private key type used in test
const TestKeyType = utils.KeyTypeECDSA

func startModule(t *testing.T) (*module.CertsModule, func(), error) {
	testCertDir := filepath.Join(os.TempDir(), "hiveot-certs-test")

	// clea start
	_ = os.RemoveAll(TestCertDir)

	m := module.NewCertsModule(testCertDir)
	err := m.Start("")
	require.NoError(t, err)
	return m, func() {
		m.Stop()
	}, err
}

// TestMain create a test folder for certificates and private key
func TestMain(m *testing.M) {
	TestCertDir = filepath.Join(os.TempDir(), "hiveot-certs-test")

	utils.SetLogging("info", "")

	result := m.Run()
	if result != 0 {
		println("Test failed with code:", result)
		println("Find test files in:", TestCertDir)
	} else {
		// comment out the next line to be able to inspect results
		_ = os.RemoveAll(TestCertDir)
	}

	os.Exit(result)
}

// Generic store store testcases
func TestStartStop(t *testing.T) {
	t.Logf("---%s---\n", t.Name())

	m, stopFn, err := startModule(t)
	_ = m
	require.NoError(t, err)
	defer stopFn()
}

func TestX509ToFromPem(t *testing.T) {
	testCerts := selfsigned.CreateTestCertBundle(TestKeyType)
	asPem := certutils.X509CertToPEM(testCerts.CaCert)
	assert.NotEmpty(t, asPem)
	asX509, err := certutils.X509CertFromPEM(asPem)
	assert.NoError(t, err)
	assert.NotEmpty(t, asX509)
}

func TestSaveLoadX509Cert(t *testing.T) {
	// hostnames := []string{"localhost"}
	caPemFile := path.Join(TestCertDir, "caCert.pem")

	testCerts := selfsigned.CreateTestCertBundle(TestKeyType)

	// save the test x509 cert
	// FIXME: this CA is created with a different private key
	err := certutils.SaveX509CertToPEM(testCerts.CaCert, caPemFile)
	assert.NoError(t, err)

	caCert, err := certutils.LoadX509CertFromPEM(caPemFile)
	assert.NoError(t, err)
	assert.NotNil(t, caCert)

	// create a server TLS cert
	tlsCert := certutils.X509CertToTLS(caCert, testCerts.CaPrivKey)
	assert.NotEmpty(t, tlsCert)

}

func TestPublicKeyFromCert(t *testing.T) {
	testCerts := selfsigned.CreateTestCertBundle(TestKeyType)
	kt, pubKey := certutils.PublicKeyFromCert(testCerts.CaCert)
	assert.NotEmpty(t, pubKey)
	assert.NotEmpty(t, kt)
}

func TestSaveLoadTLSCert(t *testing.T) {
	// hostnames := []string{"localhost"}
	certFile := path.Join(TestCertDir, "x509cert.pem")
	keyFile := path.Join(TestCertDir, "tlskey.pem")

	testCerts := selfsigned.CreateTestCertBundle(TestKeyType)

	// save the test x509 part of the TLS cert
	err := certutils.SaveTLSCertToPEM(testCerts.ServerCert, certFile, keyFile)
	assert.NoError(t, err)

	// load back the x509 part of the TLS cert
	cert, err := certutils.LoadTLSCertFromPEM(certFile, keyFile)
	assert.NoError(t, err)
	assert.NotNil(t, cert)
}

func TestService(t *testing.T) {
	m, cancelFn, err := startModule(t)
	_ = m
	require.NoError(t, err)
	defer cancelFn()

	caCert, err := m.GetCACert()
	require.NoError(t, err)
	require.NotEmpty(t, caCert)

	tlsServerCert, err := m.GetDefaultServerTlsCert()
	require.NoError(t, err)
	require.NotEmpty(t, tlsServerCert)

	serverCert, serverKey, err := certutils.TLSCertToX509(tlsServerCert)
	require.NoError(t, err)
	require.NotEmpty(t, serverKey)

	err = m.VerifyCert(certs.DefaultServerName, serverCert)
	require.NoError(t, err)
}

func TestMsgClient(t *testing.T) {
	m, cancelFn, err := startModule(t)
	_ = m
	require.NoError(t, err)
	defer cancelFn()

	// use a direct transport instead of running a client-server
	tp := direct.NewDirectTransport("testclient", m)
	cl := certsclient.NewCertsMsgClient(certs.DefaultCertsThingID, tp)
	caCert, err := cl.GetCACert()
	require.NoError(t, err)
	require.NotEmpty(t, caCert)

	modCA, _ := m.GetCACert()
	cn, err := selfsigned.VerifyCert(modCA, caCert)
	assert.NoError(t, err)
	assert.NotEmpty(t, cn)

	// assert.Equal(t, testCerts.CaCert.Issuer, caCert.Issuer.CommonName)
	// var _ certs.ICertsService = cl // interface check
	_ = cl
}

func TestCreateCerts(t *testing.T) {
	m, cancelFn, err := startModule(t)
	_ = m
	require.NoError(t, err)
	defer cancelFn()

	caCert, caPrivKey, err := m.CreateCACert()
	require.NoError(t, err)
	require.NotNil(t, caCert)
	require.NotNil(t, caPrivKey)

	serverTlsCert, err := m.CreateServerCert("test", "hostname", nil, nil)
	require.NoError(t, err)
	require.NotNil(t, serverTlsCert)

	// this needs completion
	cl := certsclient.NewCertsMsgClient(certs.DefaultCertsThingID, nil)
	// var _ certs.ICertsService = cl // interface check
	_ = cl
}

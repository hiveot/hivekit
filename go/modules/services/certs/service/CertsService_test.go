package service_test

import (
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hiveot/hivekit/go/lib/logging"
	"github.com/hiveot/hivekit/go/modules/services/certs"
	"github.com/hiveot/hivekit/go/modules/services/certs/service"
	"github.com/hiveot/hivekit/go/modules/services/certs/service/selfsigned"
)

var TestCertDir string

// TestMain create a test folder for certificates and private key
func TestMain(m *testing.M) {
	TestCertDir = filepath.Join(os.TempDir(), "hiveot-certs-test")

	logging.SetLogging("info", "")

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

func TestX509ToFromPem(t *testing.T) {
	testCerts := selfsigned.CreateTestCertBundle()
	asPem := service.X509CertToPEM(testCerts.CaCert)
	assert.NotEmpty(t, asPem)
	asX509, err := service.X509CertFromPEM(asPem)
	assert.NoError(t, err)
	assert.NotEmpty(t, asX509)
}

func TestSaveLoadX509Cert(t *testing.T) {
	// hostnames := []string{"localhost"}
	caPemFile := path.Join(TestCertDir, "caCert.pem")

	testCerts := selfsigned.CreateTestCertBundle()

	// save the test x509 cert
	// FIXME: this CA is created with a different private key
	err := service.SaveX509CertToPEM(testCerts.CaCert, caPemFile)
	assert.NoError(t, err)

	caCert, err := service.LoadX509CertFromPEM(caPemFile)
	assert.NoError(t, err)
	assert.NotNil(t, caCert)

	// create a server TLS cert
	tlsCert := service.X509CertToTLS(caCert, testCerts.CaKey)
	assert.NotEmpty(t, tlsCert)

}

func TestPublicKeyFromCert(t *testing.T) {
	testCerts := selfsigned.CreateTestCertBundle()
	pubKey := service.PublicKeyFromCert(testCerts.CaCert)
	assert.NotEmpty(t, pubKey)
}

func TestSaveLoadTLSCert(t *testing.T) {
	// hostnames := []string{"localhost"}
	certFile := path.Join(TestCertDir, "x509cert.pem")
	keyFile := path.Join(TestCertDir, "tlskey.pem")

	testCerts := selfsigned.CreateTestCertBundle()

	// save the test x509 part of the TLS cert
	err := service.SaveTLSCertToPEM(testCerts.ServerCert, certFile, keyFile)
	assert.NoError(t, err)

	// load back the x509 part of the TLS cert
	cert, err := service.LoadTLSCertFromPEM(certFile, keyFile)
	assert.NoError(t, err)
	assert.NotNil(t, cert)
}

func TestService(t *testing.T) {
	svc := service.NewCertsService(TestCertDir)
	err := svc.Start()
	require.NoError(t, err)

	caCert, err := svc.GetCACert()
	require.NoError(t, err)
	require.NotEmpty(t, caCert)

	tlsServerCert, err := svc.GetDefaultServerCert()
	require.NoError(t, err)
	require.NotEmpty(t, tlsServerCert)

	serverCert, serverKey, err := service.TLSCertToX509(tlsServerCert)
	require.NoError(t, err)
	require.NotEmpty(t, serverKey)

	err = svc.VerifyCert(certs.DefaultServerName, serverCert)
	require.NoError(t, err)

	svc.Stop()
}

package certstest_test

import (
	"crypto/x509"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/cells/certs"
	certsclient "github.com/hiveot/hivekit/go/cells/certs/client"
	certsservice "github.com/hiveot/hivekit/go/cells/certs/service"
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
	cfg := &certs.CertsConfig{CertsDir: storageDir}
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

func TestService(t *testing.T) {
	const appName = "server"

	m, cancelFn, err := startService(t)
	_ = m
	require.NoError(t, err)
	defer cancelFn()

	caCert := m.GetCACert()
	require.NotEmpty(t, caCert)

	// create the server cert
	privKey, pubKey := utils.NewEd25519Key()
	_ = privKey
	newServerCert, err := m.CreateServerCert(appName, "", time.Hour, pubKey)
	require.NoError(t, err)
	require.NotEmpty(t, newServerCert)

	certChain, err := m.GetServerCert(appName)
	require.NoError(t, err)
	require.NotEmpty(t, certChain)

	//
	caPool := x509.NewCertPool()
	caPool.AddCert(caCert)
	opts := x509.VerifyOptions{
		Roots:     caPool,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	_, err = certChain[0].Verify(opts)
	require.NoError(t, err)
}

func TestCertClient(t *testing.T) {
	const clientID = "clientID"

	m, cancelFn, err := startService(t)
	_ = m
	require.NoError(t, err)
	defer cancelFn()

	// use a direct transport instead of running a client-server
	tp := testenv.NewTestTransport(clientID, m)
	cl := certsclient.NewCertsClient(tp, "")

	privKey, pubKey := utils.NewEd25519Key()
	_ = privKey
	clientCert, err := m.CreateClientCert(clientID, api.ClientOUConsumer, time.Hour, pubKey)
	require.NoError(t, err)

	err = m.VerifyClientCert(clientID, clientCert)
	assert.NoError(t, err)

	err = cl.VerifyClientCert(clientID, clientCert)
	assert.NoError(t, err)

}

func TestCreateCerts(t *testing.T) {
	m, cancelFn, err := startService(t)
	_ = m
	require.NoError(t, err)
	defer cancelFn()

	caCert := m.GetCACert()
	require.NotNil(t, caCert)

	privKey, pubKey := utils.NewEd25519Key()
	_ = privKey
	serverChain, err := m.CreateServerCert("test", "hostname", time.Hour, pubKey)
	require.NoError(t, err)
	require.NotNil(t, serverChain)

	// this needs completion
	cl := certsclient.NewCertsClient(nil, "")
	// var _ certs.ICertsService = cl // interface check
	_ = cl
}

package certstest_test

import (
	"crypto/x509"
	"os"
	"testing"
	"time"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/cells/certs"
	certs_client "github.com/hiveot/hivekit/go/cells/certs/client"
	certs_service "github.com/hiveot/hivekit/go/cells/certs/service"
	"github.com/hiveot/hivekit/go/testenv"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// var storageDir = filepath.Join(testenv.TestHome, "certs")

// private key type used in test
const TestKeyType = utils.KeyTypeECDSA

// start the certs service
func startService(t *testing.T, certValidity time.Duration) (certs.ICertsService, error) {

	// clean start
	os.RemoveAll(testenv.TestHome)
	env := api.NewAppEnvironment(testenv.TestHome, false)
	cfg := &certs.CertsConfig{
		AdminCertValidity:  certValidity,
		CACertValidity:     certValidity,
		ServerCertValidity: certValidity,
	}
	svc, err := certs_service.NewCertsService(env, cfg)
	require.NoError(t, err)
	return svc, err
}

// TestMain create a test folder for certificates and private key
func TestMain(m *testing.M) {

	utils.SetLogging("info", "")

	result := m.Run()
	if result != 0 {
		println("Test failed with code:", result)
		// println("Find test files in:", storageDir)
	} else {
		// comment out the next line to be able to inspect results
		// _ = os.RemoveAll(storageDir)
	}

	os.Exit(result)
}

// Generic store store testcases
func TestStartStop(t *testing.T) {
	t.Logf("---%s---\n", t.Name())

	svc, err := startService(t, time.Minute)
	require.NoError(t, err)
	defer svc.Stop()
}

// verify proper self-signed CA creation
func TestCA(t *testing.T) {
	t.Logf("---%s---\n", t.Name())

	svc, err := startService(t, time.Minute)
	require.NoError(t, err)
	defer svc.Stop()

	caCert := svc.GetCACert()
	require.NotNil(t, caCert)

	// CA must be valid for less than a minute
	maxValid := time.Now().Add(time.Minute)
	assert.Less(t, caCert.NotAfter, maxValid)

}

func TestCertClient(t *testing.T) {
	const clientID = "clientID"
	var certsThingID string
	privKey := utils.NewEd25519Key()

	svc, err := startService(t, time.Minute)
	require.NoError(t, err)
	defer svc.Stop()

	// first test with a directly created cert
	clientCert1, err := svc.CreateClientCert(
		clientID, api.ClientRoleViewer, time.Hour, privKey.Public())
	require.NoError(t, err)
	err = svc.VerifyClientCert(clientID, clientCert1)
	assert.NoError(t, err)

	// use a direct transport instead of running a client-server
	certsThingID = svc.GetID()
	tp := testenv.NewTestTransport(clientID, svc)
	cl := certs_client.NewCertsClient(certsThingID, tp)
	clientCert2, err := cl.CreateClientCert(clientID, api.ClientRoleViewer, time.Hour, privKey.Public())
	require.NoError(t, err)
	err = svc.VerifyClientCert(clientID, clientCert2)
	assert.NoError(t, err)
}

func TestServerCert(t *testing.T) {
	const appName = "server"

	// starting the server initializes the CA and server cert
	svc, err := startService(t, time.Minute)
	require.NoError(t, err)
	defer svc.Stop()

	serverCert, err := svc.GetServerCert("")
	require.NoError(t, err)
	require.NotEmpty(t, serverCert)

	// force a renewal of the server cert
	err = svc.RefreshServerCert(appName, time.Hour, time.Hour)
	require.NoError(t, err)

	// the validity of the new cert must be just under an hour
	serverCert2, err := svc.GetServerCert("")
	require.NoError(t, err)
	validFor := serverCert2[0].NotAfter.Sub(time.Now())
	assert.Less(t, validFor, time.Hour)
	assert.Greater(t, validFor, time.Minute*50)

	err = svc.RefreshServerCert(appName, time.Hour, time.Hour)
	require.NoError(t, err)

	// verify the cert
	caCert := svc.GetCACert()
	caPool := x509.NewCertPool()
	caPool.AddCert(caCert)
	opts := x509.VerifyOptions{
		Roots:     caPool,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	_, err = serverCert2[0].Verify(opts)
	require.NoError(t, err)
}

func TestCreateCerts(t *testing.T) {
	svc, err := startService(t, time.Minute)
	require.NoError(t, err)
	defer svc.Stop()

	caCert := svc.GetCACert()
	require.NotNil(t, caCert)

	privKey := utils.NewEd25519Key()
	serverChain, err := svc.CreateServerCert(
		"test", "hostname", time.Hour, privKey.Public())
	require.NoError(t, err)
	require.NotNil(t, serverChain)

	// this needs completion
	cl := certs_client.NewCertsClient(svc.GetID(), nil)

	// var _ certs.ICertsService = cl // interface check
	_ = cl
}

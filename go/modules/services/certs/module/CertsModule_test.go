package module_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hiveot/hivekit/go/modules/services/certs/module"
	"github.com/stretchr/testify/require"
)

func startModule(t *testing.T) (*module.CertsModule, func(), error) {
	testCertDir := filepath.Join(os.TempDir(), "hiveot-certs-test")

	m := module.NewCertsModule(testCertDir)
	err := m.Start()
	require.NoError(t, err)
	return m, func() {
		m.Stop()
	}, err
}

// Generic store store testcases
func TestStartStop(t *testing.T) {
	t.Logf("---%s---\n", t.Name())

	m, stopFn, err := startModule(t)
	_ = m
	require.NoError(t, err)
	defer stopFn()
}

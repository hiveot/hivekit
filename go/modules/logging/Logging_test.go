package logging_test

import (
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"

	"github.com/hiveot/hivekit/go/modules/logging"
	"github.com/hiveot/hivekit/go/modules/logging/config"
	"github.com/hiveot/hivekit/go/modules/logging/module"
	"github.com/hiveot/hivekit/go/msg"
	"github.com/stretchr/testify/require"
)

// store location for logging
var LogFile = path.Join(os.TempDir(), "hivekit/logs/module1.log")

// Test creating and deleting the history database
// This requires a local unsecured MongoDB instance
func TestStartStop(t *testing.T) {
	t.Logf("---%s---\n", t.Name())

	os.RemoveAll(filepath.Dir(LogFile))
	cfg := config.NewLoggingConfig(LogFile, logging.LoggingBackendFile)
	m := module.NewLoggingModule(cfg)
	err := m.Start("")
	require.NoError(t, err)
	m.Stop()
}

func TestLogNotification(t *testing.T) {
	t.Logf("---%s---\n", t.Name())

	//setup
	os.RemoveAll(filepath.Dir(LogFile))
	cfg := config.NewLoggingConfig(LogFile, logging.LoggingBackendFile)
	cfg.Log2Stdout = true
	m := module.NewLoggingModule(cfg)
	err := m.Start("")
	require.NoError(t, err)
	defer m.Stop()

	//pass events through the module and log them in a file destination
	ev1 := msg.NewNotificationMessage("agent1", msg.AffordanceTypeEvent, "thing1", "name1", nil)
	m.HandleNotification(ev1)

	// wait for write to log to complete
	time.Sleep(time.Millisecond * 10)
}

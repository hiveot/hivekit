package utils_test

import (
	"log/slog"
	"os"
	"testing"

	"github.com/hiveot/hivekit/go/utils"
)

func TestLogging(t *testing.T) {
	//wd, _ := os.Getwd()
	//logFile := path.Join(wd, "../../test/logs/TestLogging.log")
	logFile := ""

	os.Remove(logFile)
	utils.SetLogging("info", logFile)
	slog.Info("Hello info")
	utils.SetLogging("debug", logFile)
	slog.Debug("Hello debug")
	utils.SetLogging("warn", logFile)
	slog.Warn("Hello warn")
	utils.SetLogging("error", logFile)
	slog.Error("Hello error")
	//assert.FileExists(t, logFile)
	//os.Remove(logFile)
}

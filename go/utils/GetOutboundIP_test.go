package utils_test

import (
	"log/slog"
	"testing"

	"github.com/hiveot/hivekit/go/utils"
	"github.com/stretchr/testify/assert"
)

func TestGetOutboundIP(t *testing.T) {
	addr := utils.GetOutboundIP("")
	assert.NotEmpty(t, addr)
	slog.Info("TestGetOutboundIP", "addr", addr)
}

func TestGetOutboundIPBadAddr(t *testing.T) {
	addr := utils.GetOutboundIP("badaddress")
	assert.Empty(t, addr)
}

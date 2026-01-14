package utils_test

import (
	"log/slog"
	"testing"

	"github.com/hiveot/hivekit/go/utils"
	"github.com/stretchr/testify/assert"
)

func TestGetOutboundInterface(t *testing.T) {
	name, mac, addr := utils.GetOutboundInterface("")
	assert.NotEmpty(t, name)
	assert.NotEmpty(t, mac)
	assert.NotEmpty(t, addr)
	slog.Info("TestGetOutboundInterface", "name", name, "mac", mac, "addr", addr)
}

func TestGetOutboundInterfaceBadAddr(t *testing.T) {
	name, _, _ := utils.GetOutboundInterface("badaddress")
	assert.Empty(t, name)
}

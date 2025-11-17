package net_test

import (
	"log/slog"
	"testing"

	"github.com/hiveot/gocore/utils/net"
	"github.com/stretchr/testify/assert"
)

func TestGetOutboundInterface(t *testing.T) {
	name, mac, addr := net.GetOutboundInterface("")
	assert.NotEmpty(t, name)
	assert.NotEmpty(t, mac)
	assert.NotEmpty(t, addr)
	slog.Info("TestGetOutboundInterface", "name", name, "mac", mac, "addr", addr)
}

func TestGetOutboundInterfaceBadAddr(t *testing.T) {
	name, _, _ := net.GetOutboundInterface("badaddress")
	assert.Empty(t, name)
}

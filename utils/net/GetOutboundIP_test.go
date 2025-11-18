package net_test

import (
	"log/slog"
	"testing"

	"github.com/hiveot/hivekitgo/utils/net"
	"github.com/stretchr/testify/assert"
)

func TestGetOutboundIP(t *testing.T) {
	addr := net.GetOutboundIP("")
	assert.NotEmpty(t, addr)
	slog.Info("TestGetOutboundIP", "addr", addr)
}

func TestGetOutboundIPBadAddr(t *testing.T) {
	addr := net.GetOutboundIP("badaddress")
	assert.Empty(t, addr)
}

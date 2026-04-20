package authnpkg

import (
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/authn/internal/httpauthn"
	"github.com/hiveot/hivekit/go/modules/transports"
)

// Create a module instance for handling authn requests over http
// Intended for supporting user requests such as login, logout, and refreshToken.
//
// This module provides passthrough for all requests and responses and injects new
// requests received over http. The authn module must be installed downstream to handle
// these requests.
func NewAuthnUserHttpService(httpServer transports.IHttpServer) modules.IHiveModule {
	m := httpauthn.NewAuthnUserHttpService(httpServer)
	return m
}

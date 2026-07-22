package authn_service

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/modules/authn/internal/httpapi"
)

// Create a module instance for handling authn requests over http
// Intended for supporting user requests such as login, logout, and refreshToken.
//
// This module provides passthrough for all requests and responses and injects new
// requests received over http. The authn module must be installed downstream to handle
// these requests.
func NewAuthnUserHttpService(httpServer api.IHttpServer) api.IHiveModule {
	m := httpapi.NewAuthnUserHttpService(httpServer)
	return m
}

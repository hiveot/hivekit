package authn_service

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/cells/authn/internal/httpapi"
)

// Create a cell instance for handling authn requests over HTTP.
// Intended for supporting user requests such as login, logout, and refreshToken.
//
// This provides passthrough for all requests and responses, and injects new requests
// received over http. The authn service must be installed downstream to handle
// these requests.
func NewAuthnUserHttpService(httpServer api.IHttpServer) api.IHiveCell {
	m := httpapi.NewAuthnUserHttpService(httpServer)
	return m
}

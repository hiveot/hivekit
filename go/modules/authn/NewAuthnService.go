package authn

import (
	authnapi "github.com/hiveot/hivekit/go/modules/authn/api"

	"github.com/hiveot/hivekit/go/modules/authn/internal/service"
	"github.com/hiveot/hivekit/go/modules/transports"
)

// NewAuthnService is the factory function to create a new instance of the
// authentication server side module.
//
// Note 1: Currently, only a single instance of this module can be used as the
// thingID's of the module services are fixed.
//
// Note 2: to avoid a chicken-and-egg problem between authentication and http server,
// create the http server first and pass it to the authenticator. The authenticator will
// invoke httpserver.SetAuthValidator on start.
//
// authnConfig contains the password storage and token management configuration
// httpServer to server the http endpoint or nil to not use http.
func NewAuthnService(
	authnConfig authnapi.AuthnConfig,
	httpServer transports.IHttpServer) authnapi.IAuthnService {

	m := service.NewAuthnService(authnConfig, httpServer)
	return m
}

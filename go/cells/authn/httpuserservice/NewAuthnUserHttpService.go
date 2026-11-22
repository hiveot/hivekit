package authn_httpuserservice

import (
	"errors"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/cells/authn"
	"github.com/hiveot/hivekit/go/cells/authn/httpuserservice/internal"
)

// Start the service for handling authn requests over HTTP.
// Intended for supporting user requests such as login, logout, and refreshToken.
//
// This forwards auth requests to the authn user service downstream.
func NewAuthnUserHttpService(authnServiceID string, httpServer api.IHttpServer) api.IHiveCell {
	svc := internal.NewAuthnUserHttpServiceImpl(authnServiceID, httpServer)
	return svc
}

// Return a ready-to-use instance of the http authentication server using the factory environment.
func NewAuthnUserHttpServiceFactory(
	f api.ICellFactory, md *api.CellDefinition) (api.IHiveCell, error) {

	// don't access the cell directly.
	// Forwarding the request allows the use of an isolated sandbox.
	// authnSvc := f.GetCell(authn.AuthnUserServiceCellType)

	authnServiceID := authn.AuthnUserServiceCellType
	httpServer := f.GetHttpServer(true)
	if httpServer == nil {
		return nil, errors.New("NewAuthnUserHttpServiceFactory: Need a http server but factory cant provide it")
	}
	svc := NewAuthnUserHttpService(authnServiceID, httpServer)
	return svc, nil
}

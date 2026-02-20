package tptests

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/transports"
)

// A dummy transport for testing
// This implements IHttpServer and ITransportModule interfaces
type DummyServer struct {
	modules.HiveModuleBase

	url           string
	protRoute     chi.Router
	pubRoute      chi.Router
	validateToken transports.ValidateTokenHandler
}

func (d *DummyServer) GetConnectURL() string {
	return d.url
}

func (d *DummyServer) GetClientIdFromContext(r *http.Request) (string, string, error) {
	return "", "", errors.New("not implemented")
}

func (d *DummyServer) GetRequestParams(r *http.Request) (transports.RequestParams, error) {
	rp := transports.RequestParams{}
	return rp, fmt.Errorf("not supported in dummy server")
}
func (d *DummyServer) GetProtectedRoute() chi.Router {
	return d.protRoute
}
func (d *DummyServer) GetPublicRoute() chi.Router {
	return d.pubRoute
}
func (d *DummyServer) SetAuthValidator(validator transports.ValidateTokenHandler) {
	d.validateToken = validator
}
func NewDummyServer(url string) transports.IHttpServer {
	rootRouter := chi.NewRouter()
	rootRouter.Use(middleware.Heartbeat(transports.DefaultPingPath))
	d := &DummyServer{
		url:       url,
		protRoute: rootRouter.With(),
		pubRoute:  rootRouter.With(),
	}
	return d
}

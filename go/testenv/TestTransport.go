package testenv

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
// This implements IHttpServer and ITransportServer interfaces
type TestTransport struct {
	modules.HiveModuleBase

	url       string
	protRoute chi.Router
	pubRoute  chi.Router
	authr     transports.IAuthenticator
}

func (d *TestTransport) GetAuthenticator() transports.IAuthenticator {
	return d.authr
}

func (d *TestTransport) GetConnectURL() string {
	return d.url
}

func (d *TestTransport) GetClientIdFromContext(r *http.Request) (string, error) {
	return "", errors.New("not implemented")
}

func (d *TestTransport) GetRequestParams(r *http.Request) (transports.RequestParams, error) {
	rp := transports.RequestParams{}
	return rp, fmt.Errorf("not supported in dummy server")
}
func (d *TestTransport) GetProtectedRoute() chi.Router {
	return d.protRoute
}
func (d *TestTransport) GetPublicRoute() chi.Router {
	return d.pubRoute
}
func (d *TestTransport) SetAuthenticator(authr transports.IAuthenticator) {
	d.authr = authr
}
func (d *TestTransport) Start() error {
	return nil
}
func (d *TestTransport) Stop() {
}

func NewDummyServer(url string) transports.IHttpServer {
	rootRouter := chi.NewRouter()
	rootRouter.Use(middleware.Heartbeat(transports.DefaultPingPath))
	d := &TestTransport{
		url:       url,
		protRoute: rootRouter.With(),
		pubRoute:  rootRouter.With(),
	}
	return d
}

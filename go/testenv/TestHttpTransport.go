package testenv

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/transport"
)

// A dummy transport for testing
// This implements IHttpServer and ITransportServer interfaces
type TestHttpTransport struct {
	modules.HiveModuleBase

	url       string
	protRoute chi.Router
	pubRoute  chi.Router
	authr     transport.IAuthenticator
}

func (d *TestHttpTransport) GetAuthenticator() transport.IAuthenticator {
	return d.authr
}

func (d *TestHttpTransport) GetConnectURL() string {
	return d.url
}

func (d *TestHttpTransport) GetClientIdFromContext(r *http.Request) (string, error) {
	return "", errors.New("not implemented")
}

func (d *TestHttpTransport) GetRequestParams(r *http.Request) (transport.RequestParams, error) {
	rp := transport.RequestParams{}
	return rp, fmt.Errorf("not supported in dummy server")
}
func (d *TestHttpTransport) GetProtectedRoute() chi.Router {
	return d.protRoute
}
func (d *TestHttpTransport) GetPublicRoute() chi.Router {
	return d.pubRoute
}
func (d *TestHttpTransport) SetAuthenticator(authr transport.IAuthenticator) {
	d.authr = authr
}
func (d *TestHttpTransport) Start() error {
	return nil
}
func (d *TestHttpTransport) Stop() {
}

func NewDummyServer(url string) transport.IHttpServer {
	rootRouter := chi.NewRouter()
	rootRouter.Use(middleware.Heartbeat(transport.DefaultPingPath))
	d := &TestHttpTransport{
		url:       url,
		protRoute: rootRouter.With(),
		pubRoute:  rootRouter.With(),
	}
	return d
}

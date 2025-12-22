package httpserver

import "github.com/go-chi/chi/v5"

// A dummy http server for testing
// This implements IHttpServer
type DummyServer struct {
	url       string
	protRoute chi.Router
	pubRoute  chi.Router
}

func (d *DummyServer) GetConnectURL() string {
	return d.url
}
func (d *DummyServer) GetProtectedRoutes() chi.Router {
	return d.protRoute
}
func (d *DummyServer) GetPublicRoutes() chi.Router {
	return d.pubRoute
}

func NewDummyServer(url string) IHttpServer {
	d := &DummyServer{
		url:       url,
		protRoute: chi.NewRouter(),
		pubRoute:  chi.NewRouter(),
	}
	return d
}

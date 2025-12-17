package tls

import (
	"github.com/go-chi/chi/v5"
)

const DefaultTlsThingID = "tls-server"

// The default listening port if none is set
const DefaultTlsPort = 8444

// TLS server transport interface
type ITLSTransport interface {
	// Return the router used by the TLS service.
	// Intended to let services add their endpoints.
	//
	// Local use only. nil when queried remotely.
	GetRouter() *chi.Mux
}

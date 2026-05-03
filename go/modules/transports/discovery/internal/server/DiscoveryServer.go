package internal

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/grandcat/zeroconf"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/directory"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/modules/transports/discovery"
)

// DiscoveryServer is a module for serving a TD over http and publishing a corresponding
// DNS-SD service record.
// Use DiscoveryClient for discovering devices on the network.
type DiscoveryServer struct {
	modules.HiveModuleBase

	// this instance thingID
	discoveryThingID string

	// the directory thingID the intercept for discovery of TD
	directoryThingID string

	// optional additional endpoints to publish in the discovery record in addition to
	// the well-known exploration URL.
	endpoints map[string]string

	// service discovery using mDNS
	dnssdServer *zeroconf.Server

	// the http server that servers the exploration endpoint.
	httpServer transports.IHttpServer
}

// ServeDirectoryTD registers the given directory TD with the http server
// and publishes its endpoint using DNS-SD discovery.
//
// dirTDJSON must be provided by a directory that implements the affordances.
//
// If a list of transports is available this updates the TD security scheme,
// base URL and forms.
//
// This aims to be compliant with https://w3c.github.io/wot-discovery/#exploration-server
//
// This fails if the http server isn't provided.
func (m *DiscoveryServer) ServeDirectoryTD(dirTDJSON string) (err error) {
	// map of endpoints by scheme (wss, sse, ...)

	if m.dnssdServer != nil {
		return fmt.Errorf("ServeDirectoryTDD: a TD is already served")
	}
	publicRoute := m.httpServer.GetPublicRoute()
	// TBD: support for base path?
	wellKnownPath := directory.WellKnownWoTPath

	publicRoute.Get(wellKnownPath, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/td+json")
		_, _ = w.Write([]byte(dirTDJSON))
	})
	instanceName := m.discoveryThingID
	tddURL, err := url.JoinPath(m.httpServer.GetConnectURL(), wellKnownPath)

	m.dnssdServer, err = ServeWotDiscovery(
		instanceName, tddURL, discovery.WOT_DIRECTORY_SERVICE_TYPE, m.endpoints)

	if err != nil {
		slog.Error("Failed starting introduction server for DNS-SD",
			"TDD URL", tddURL,
			"err", err.Error())
		return err
	}
	return nil
}

// ServeThingTD registers the given thing TD with the http server
// and publishes its endpoint using DNS-SD discovery.
// Indended for use by things that run servers. (not recommended)
func (m *DiscoveryServer) ServeThingTD(thingTDJSON string) (err error) {

	slog.Info("DiscoveryServer. Serving Thing TD")

	if m.dnssdServer != nil {
		return fmt.Errorf("ServiceThingTD: a TD is already served")
	}

	publicRoute := m.httpServer.GetPublicRoute()
	// TBD: support for base path?
	wellKnownPath := directory.WellKnownWoTPath
	publicRoute.Get(wellKnownPath, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(thingTDJSON))
	})
	instanceName := m.discoveryThingID
	thingTDURL, err := url.JoinPath(m.httpServer.GetConnectURL(), wellKnownPath)
	m.dnssdServer, err = ServeWotDiscovery(instanceName, thingTDURL, discovery.WOT_THING_SERVICE_TYPE, nil)
	if err != nil {
		slog.Error("Failed starting introduction server for DNS-SD",
			"Thing TD URL", thingTDURL,
			"err", err.Error())
		return err
	}
	return nil
}

// Start starts the discovery module.
// This does nothing until ServeThingTD or ServeDirectoryTDD is called.
func (m *DiscoveryServer) Start() (err error) {

	slog.Info("Start: Starting discovery transport server")
	return nil
}

// Stop any running services and release resources
func (m *DiscoveryServer) Stop() {
	slog.Info("Stop: Stopping discovery transport server")
	if m.dnssdServer != nil {
		m.dnssdServer.Shutdown()
		// the DNS server takes a wee bit of time to really stop
		// Wait this wee bit to prevent a race running tests
		time.Sleep(time.Millisecond)
	}
}

// NewDiscoveryServer creates a new discovery server module instance.
//
// The serviceID is optional and defaults to DefaultDiscoveryThingID.
//
//	serviceID is the thingID of this server itself. Used as the serviceID in DNS-SD records
//	httpServer is the server that serves the TD on the well-known endpoint.
//	transports for TD security scheme, base URL and forms. Optional.
func NewDiscoveryServer(serviceID string,
	httpServer transports.IHttpServer, endpoints map[string]string) *DiscoveryServer {

	if serviceID == "" {
		serviceID = discovery.DefaultDiscoveryThingID
	}
	m := &DiscoveryServer{
		directoryThingID: directory.DefaultDirectoryThingID,
		discoveryThingID: serviceID,
		endpoints:        endpoints,
		httpServer:       httpServer,
	}
	var _ discovery.IDiscoveryServer = m // interface check
	return m
}

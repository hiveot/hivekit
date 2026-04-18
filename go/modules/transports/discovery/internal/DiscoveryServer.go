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

// DiscoveryModule is a module for serving a directory endpoint and discovering
// network devices. It serves two roles: one to publish a directory endpoint using
// mDNS, and two, to discovery devices on the network.

type DiscoveryServer struct {
	modules.HiveModuleBase

	// this instance thingID
	discoveryThingID string

	// optional additional endpoints to publish in the discovery record in addition to
	// the well-known exploration URL.
	endpoints map[string]string

	// The directory TD document in JSON for serving on the well-known exploration path.
	// dirTDJSON string

	// service discovery using mDNS
	dnssdServer *zeroconf.Server

	// the http server that servers the exploration endpoint.
	httpServer transports.IHttpServer
}

// ServeDirectoryTDD registers the given directory TD with the http server
// and publishes its endpoint using DNS-SD discovery.
//
// This fails if the http server isn't provided.
func (m *DiscoveryServer) ServeDirectoryTDD(dirTDJSON string) (err error) {
	if m.dnssdServer != nil {
		return fmt.Errorf("ServeDirectoryTDD: a TD is already served")
	}
	publicRoute := m.httpServer.GetPublicRoute()
	// TBD: support for base path?
	wellKnownPath := directory.WellKnownWoTPath
	publicRoute.Get(wellKnownPath, func(w http.ResponseWriter, r *http.Request) {
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

// Start starts the http

//  1. serves the directory TD on the .well-known/wot http endpoint.
//  2. publishes a DNS-SD record of the directory TD with the service name "_directory._sub._wot._tcp".
//     containing a TXT record for 'td', 'type', 'scheme' as described in
//     https://w3c.github.io/wot-discovery/#introduction-dns-sd-sec
//  3. start listening for devices and publish notifications on discovered devices
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
//	httpServer is the server that serves the TD on the well-known endpoint.
//	endpoints are optional additional URLS to include in the DNS-SD discovery record
//	 where key is the schema "http", "wss", "sse-sc" and value the URL.
//	thingID is the service
func NewDiscoveryServer(
	httpServer transports.IHttpServer, endpoints map[string]string, serviceID string) *DiscoveryServer {

	if serviceID == "" {
		serviceID = discovery.DefaultDiscoveryThingID
	}
	m := &DiscoveryServer{
		discoveryThingID: serviceID,
		endpoints:        endpoints,
		httpServer:       httpServer,
	}
	var _ discovery.IDiscoveryServer = m // interface check
	return m
}

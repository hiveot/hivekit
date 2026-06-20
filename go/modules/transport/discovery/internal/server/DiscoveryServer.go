package internal

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/grandcat/zeroconf"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/directory"
	"github.com/hiveot/hivekit/go/modules/transport"
	"github.com/hiveot/hivekit/go/modules/transport/discovery"
)

// DiscoveryServer is a module for serving a TD over http and publishing a corresponding
// DNS-SD service record.
// Use DiscoveryClient for discovering devices on the network.
type DiscoveryServer struct {
	*modules.HiveModuleBase

	// the directory thingID the intercept for discovery of TD
	directoryThingID string

	// optional additional endpoints to publish in the discovery record in addition to
	// the well-known exploration URL.
	endpoints map[string]string

	// service discovery using mDNS
	dnssdServer *zeroconf.Server

	// the http server that servers the exploration endpoint.
	httpServer transport.IHttpServer
}

// When a request to create/update a TD is received then serve it in discovery.
// Intended for use in a module chain where the device publishes its TD instead
// of updating a external directory.
func (m *DiscoveryServer) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
	// intercept a directory update to publish a TD
	// no need to check the directory thingID, the action name in this chain is sufficient.
	if req.Operation == td.OpInvokeAction && //req.ThingID == m.directoryThingID &&
		(req.Name == directory.CreateThingAction || req.Name == directory.UpdateThingAction) {

		tdJson := req.ToString(0)
		m.ServeThingTD(tdJson)
		resp := req.CreateResponse(nil, nil)
		replyTo(resp)
		return nil
	} else {
		return m.HiveModuleBase.HandleRequest(req, replyTo)
	}
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
		return fmt.Errorf("ServeDirectoryTD: a TD is already served")
	} else if m.httpServer == nil {
		return fmt.Errorf("ServeDirectoryTD: missing http server")
	}
	publicRoute := m.httpServer.GetPublicRoute()
	// TBD: support for base path?
	wellKnownPath := directory.WellKnownWoTPath

	publicRoute.Get(wellKnownPath, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/td+json")
		_, _ = w.Write([]byte(dirTDJSON))
	})
	instanceName := m.GetThingID()
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
	instanceName := m.GetThingID()
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
//	thingID of this server module. Also used as the serviceID in DNS-SD records
//	httpServer is the server that serves the TD on the well-known endpoint.
//	transports for TD security scheme, base URL and forms. Optional.
func NewDiscoveryServer(thingID string,
	httpServer transport.IHttpServer, endpoints map[string]string) *DiscoveryServer {

	if thingID == "" {
		thingID = discovery.DiscoveryServerModuleType
	}
	m := &DiscoveryServer{
		HiveModuleBase: modules.NewHiveModuleBase(thingID, 0),

		directoryThingID: directory.DefaultDirectoryThingID,
		endpoints:        endpoints,
		httpServer:       httpServer,
	}
	var _ discovery.IDiscoveryServer = m // interface check
	return m
}

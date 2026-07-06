package internal

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/grandcat/zeroconf"
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/directory"
	"github.com/hiveot/hivekit/go/modules/transport/discovery"
	"github.com/teris-io/shortid"
)

// DiscoveryServerImpl is a module for serving a TD over http and publishing a corresponding
// DNS-SD service record.
// Use DiscoveryClient for discovering devices on the network.
type DiscoveryServerImpl struct {
	*modules.HiveModuleBase

	// the directory thingID the intercept for discovery of TD
	directoryThingID string

	// optional additional endpoints to publish in the discovery record in addition to
	// the well-known exploration URL.
	endpoints map[string]string

	// service discovery using mDNS
	dnssdServer *zeroconf.Server

	// the http server that servers the exploration endpoint.
	httpServer api.IHttpServer
}

// Handle request to serve a directory or Thing TD.
// Intended for use in a module chain where a device or directory publishes its TD for discovery.
func (m *DiscoveryServerImpl) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {

	// no need to check the discovery thingID, the action name in this chain is sufficient.
	if req.Operation == td.OpInvokeAction && req.Name == discovery.ServeDirectoryTDAction {

		tddJson := req.ToString(0)
		err = m.ServeDirectoryTD(tddJson)
		resp := req.CreateResponse(nil, err)
		return replyTo(resp)
	} else if req.Operation == td.OpInvokeAction && req.Name == discovery.ServeThingTDAction {
		tdJson := req.ToString(0)
		err = m.ServeThingTD(tdJson)
		resp := req.CreateResponse(nil, err)
		return replyTo(resp)
	}
	return m.HiveModuleBase.HandleRequest(req, replyTo)
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
func (m *DiscoveryServerImpl) ServeDirectoryTD(dirTDJSON string) (err error) {
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

// ServeThingTD registers the given thing TD with the HTTP server and publishes
// its provisioning endpoint using DNS-SD discovery.
// Indended for use by stand-alone things that run servers.
func (m *DiscoveryServerImpl) ServeThingTD(thingTDJSON string) (err error) {

	slog.Info("DiscoveryServer. Serving Thing TD")

	if m.dnssdServer != nil {
		return fmt.Errorf("ServiceThingTD: a TD is already served")
	}

	// serve the TD on the well-known http endpoint
	publicRoute := m.httpServer.GetPublicRoute()
	wellKnownPath := directory.WellKnownWoTPath
	publicRoute.Get(wellKnownPath, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(thingTDJSON))
	})

	// publish a discovery record
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
func (m *DiscoveryServerImpl) Start() (err error) {

	slog.Info("Start: Starting discovery transport server")
	return nil
}

// Stop any running services and release resources
func (m *DiscoveryServerImpl) Stop() {
	slog.Info("Stop: Stopping discovery transport server")
	if m.dnssdServer != nil {
		m.dnssdServer.Shutdown()
		// the DNS server takes a wee bit of time to really stop
		// Wait this wee bit to prevent a race running tests
		time.Sleep(time.Millisecond)
	}
}

// NewDiscoveryServerImpl creates a new discovery server module instance.
//
// The instanceID is optional and defaults to DiscoveryServerModuleType.
//
//	instanceID of the discovery server module. This defaults to {module type}-{shortID}.
//	httpServer is the server that serves the TD on the well-known endpoint.
//	transports for TD security scheme, base URL and forms. Optional.
func NewDiscoveryServerImpl(instanceID string,
	httpServer api.IHttpServer, endpoints map[string]string) *DiscoveryServerImpl {

	if instanceID == "" {
		instanceID = discovery.DiscoveryServerModuleType + "-" + shortid.MustGenerate()
	}
	m := &DiscoveryServerImpl{
		HiveModuleBase: modules.NewHiveModuleBase(instanceID, 0),

		directoryThingID: directory.DefaultDirectoryThingID,
		endpoints:        endpoints,
		httpServer:       httpServer,
	}
	var _ discovery.IDiscoveryServer = m // interface check
	return m
}

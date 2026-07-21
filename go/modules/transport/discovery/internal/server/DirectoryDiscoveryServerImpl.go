package internal

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
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

// DirectoryDiscoveryServerImpl is a module for serving a TDD over http and publishing a corresponding
// DNS-SD service record.
// Use DiscoveryClient for discovering directories on the network.
type DirectoryDiscoveryServerImpl struct {
	*modules.HiveModuleBase

	// the directory thingID the intercept for discovery of TD
	// directoryThingID string

	// optional additional endpoints to publish in the discovery record in addition to
	// the well-known exploration URL.
	endpoints map[string]string

	// service discovery using mDNS
	dnssdServer *zeroconf.Server

	// the http server that servers the exploration endpoint.
	httpServer api.IHttpServer

	mux sync.RWMutex
}

// Handle request to serve a directory or Thing TD.
// Intended for use in a module chain where a device or directory publishes its TD for discovery.
func (m *DirectoryDiscoveryServerImpl) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {

	// no need to check the discovery thingID, the action name in this chain is sufficient.
	if req.Operation == td.OpInvokeAction {
		switch req.Name {
		case discovery.ServeDirectoryTDAction:
			tddJson := req.ToString(0)
			err = m.ServeDirectoryTD(tddJson)
			resp := req.CreateResponse(nil, err)
			return replyTo(resp)
		}
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
func (m *DirectoryDiscoveryServerImpl) ServeDirectoryTD(dirTDJSON string) (err error) {
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
		instanceName, tddURL, true, m.endpoints)

	if err != nil {
		slog.Error("Failed starting introduction server for DNS-SD",
			"TDD URL", tddURL,
			"err", err.Error())
		return err
	}
	return nil
}

// Start starts the discovery module.
//
// This waits until ServeDirectoryTD or ServeThingTD is called, or a module
// up the chain sends a corresponding action request.
func (m *DirectoryDiscoveryServerImpl) Start() (err error) {

	slog.Info("Start: Starting discovery transport server")
	return nil
}

// Stop any running services and release resources
func (m *DirectoryDiscoveryServerImpl) Stop() {
	m.mux.Lock()
	defer m.mux.Unlock()
	slog.Info("Stop: Stopping discovery transport server")
	if m.dnssdServer != nil {
		m.dnssdServer.Shutdown()
		m.dnssdServer = nil
		// the DNS server takes a wee bit of time to really stop
		// Wait this wee bit to prevent a race running tests
		time.Sleep(time.Millisecond)
	}
}

// NewDirectoryDiscoveryServerImpl creates a new discovery server module instance.
//
// The instanceID is used in publishing the discovery record. It is also the
// thingID of the module instance, and defaults to DiscoveryServerModuleType-{shortid}.
//
//	instanceID of the discovery server module. This defaults to {module type}-{shortID}.
//	httpServer is the server that serves the TD on the well-known endpoint.
//	transports for TD security scheme, base URL and forms. Optional.
func NewDirectoryDiscoveryServerImpl(instanceID string,
	httpServer api.IHttpServer, endpoints map[string]string) *DirectoryDiscoveryServerImpl {

	if instanceID == "" {
		instanceID = discovery.DirectoryDiscoveryServerModuleType + "-" + shortid.MustGenerate()
	}
	m := &DirectoryDiscoveryServerImpl{
		HiveModuleBase: modules.NewHiveModuleBase(instanceID, 0),

		// directoryThingID: directory.DefaultDirectoryThingID,
		endpoints:  endpoints,
		httpServer: httpServer,
	}
	var _ discovery.IDirectoryDiscoveryServer = m // interface check
	return m
}

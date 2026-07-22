package serverimpl

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

// ThingDiscoveryServerImpl is a module for serving a Thing TD over http and publishing a
// corresponding DNS-SD service record.
// Use DiscoveryClient for discovering devices on the network.
type ThingDiscoveryServerImpl struct {
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

// Handle request to serve a Thing TD.
// Intended for use in a module chain where a device publishes its TD for discovery.
func (m *ThingDiscoveryServerImpl) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {

	// no need to check the discovery thingID, the action name in this chain is sufficient.
	if req.Operation == td.OpInvokeAction {
		switch req.Name {
		case discovery.ServeThingTDAction:
			tdJson := req.ToString(0)
			err = m.ServeThingTD(tdJson)
			resp := req.CreateResponse(nil, err)
			return replyTo(resp)
		case directory.UpdateThingAction, directory.CreateThingAction:
			// When a device publishes their TD it is send as a create thing request.
			// With a discovery module in the chain this is used to publish a Thing discovery
			// record.
			//
			tdJson := req.ToString(0)
			err = m.ServeThingTD(tdJson)
			resp := req.CreateResponse(nil, err)
			return replyTo(resp)
		}
	}
	return m.HiveModuleBase.HandleRequest(req, replyTo)
}

// ServeThingTD registers the given thing TD with the HTTP server and publishes
// its provisioning endpoint using DNS-SD discovery.
// Indended for use by stand-alone things that run servers.
func (m *ThingDiscoveryServerImpl) ServeThingTD(thingTDJSON string) (err error) {

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
	m.dnssdServer, err = ServeWotDiscovery(instanceName, thingTDURL, false, nil)
	if err != nil {
		slog.Error("Failed starting introduction server for DNS-SD",
			"Thing TD URL", thingTDURL,
			"err", err.Error())
		return err
	}
	return nil
}

// // Start starts the discovery module.
// //
// // This waits until ServeDirectoryTD or ServeThingTD is called, or a module
// // up the chain sends a corresponding action request.
// func (m *ThingDiscoveryServerImpl) Start() (err error) {

// 	slog.Info("Start: Starting discovery transport server")
// 	return nil
// }

// Stop any running services and release resources
func (m *ThingDiscoveryServerImpl) Stop() {
	m.mux.Lock()
	defer m.mux.Unlock()
	slog.Info("Stop: Stopping Thing discovery transport server")
	if m.dnssdServer != nil {
		m.dnssdServer.Shutdown()
		m.dnssdServer = nil
		// the DNS server takes a wee bit of time to really stop
		// Wait this wee bit to prevent a race running tests
		time.Sleep(time.Millisecond)
	}
}

// NewThingDiscoveryServerImpl creates a new Thing discovery server module instance.
//
// The instanceID is used in publishing the discovery record. It is also the
// thingID of the module instance, and defaults to ThingDiscoveryServerModuleType-{shortid}.
//
//	instanceID of the discovery server module. This defaults to {module type}-{shortID}.
//	httpServer is the server that serves the TD on the well-known endpoint.
//	transports for TD security scheme, base URL and forms. Optional.
func NewThingDiscoveryServerImpl(instanceID string,
	httpServer api.IHttpServer, endpoints map[string]string) *ThingDiscoveryServerImpl {

	if instanceID == "" {
		instanceID = discovery.ThingDiscoveryServerModuleType + "-" + shortid.MustGenerate()
	}
	m := &ThingDiscoveryServerImpl{
		HiveModuleBase: modules.NewHiveModuleBase(instanceID, 0),

		// directoryThingID: directory.DefaultDirectoryThingID,
		endpoints:  endpoints,
		httpServer: httpServer,
	}
	var _ discovery.IThingDiscoveryServer = m // interface check
	return m
}

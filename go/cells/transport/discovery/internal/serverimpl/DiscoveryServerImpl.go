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
	"github.com/hiveot/hivekit/go/cells"
	"github.com/hiveot/hivekit/go/cells/directory"
	"github.com/hiveot/hivekit/go/cells/transport/discovery"
)

// DiscoveryServerImpl serves a TD over http and publishing a corresponding
// DNS-SD service record.
// This can serve a Thing TD or a Directory TD but not both.
//
// When used in a cell chain together with a directory, this service must be placed
// after the directory in the chain to prevent it from intercepting a CreateThing
// request.
//
// Use DiscoveryClient for discovering directories or things on the network.
type DiscoveryServerImpl struct {
	*cells.HiveCellBase

	// The optional directory TD to serve on start
	tddJSON string

	// optional additional endpoints to publish in the discovery record in addition to
	// the well-known exploration URL.
	endpoints map[string]string

	// service discovery using mDNS per thingID
	dnssdServers map[string]*zeroconf.Server

	// the http server that servers the exploration endpoint.
	httpServer api.IHttpServer

	mux sync.RWMutex

	// optional name under which to serve the TD(D). "" for DNS-SD provided default .
	serviceName string
}

// Handle request to serve a directory or Thing TD.
// Intended for use in a cell chain where a device or directory publishes its TD for discovery.
func (m *DiscoveryServerImpl) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {

	// no need to check the discovery thingID, the action name in this chain is sufficient.
	if req.Operation == td.OpInvokeAction {
		switch req.Name {
		case discovery.ServeDirectoryTDAction:
			tddJson := req.ToString(0)
			tddURL, err := m.ServeDirectoryTD(m.serviceName, tddJson)
			resp := req.CreateResponse(tddURL, err)
			return replyTo(resp)

		case discovery.ServeThingTDAction:
			tdJson := req.ToString(0)
			err = m.ServeThingTD("", tdJson)
			resp := req.CreateResponse(nil, err)
			return replyTo(resp)

		case directory.UpdateThingAction, directory.CreateThingAction:
			// When a device or service publishes their TD it is send as a create thing
			// request.
			// With a discovery server in the chain this is used to publish a Thing
			// discovery record. If the chain also contains a directory then the directory
			// MUST be placed before the discovery service to avoid it intercepting of the
			// request.
			tdJson := req.ToString(0)
			err = m.ServeThingTD("", tdJson)
			resp := req.CreateResponse(nil, err)
			return replyTo(resp)
		}
	}
	return m.HiveCellBase.HandleRequest(req, replyTo)
}

// ServeDirectoryTD registers the given directory TD with the http server
// and publishes its endpoint using DNS-SD discovery.
//
// This can be invoked directly of via a ServeDirectoryTDAction request.
//
//	serviceName is optional for searching for specific directory instances
//	tddJSON must be provided by a directory that implements the affordances.
//
// If a list of transports is available this updates the TD security scheme,
// base URL and forms.
//
// This aims to be compliant with https://w3c.github.io/wot-discovery/#exploration-server
//
// This returns the TDD URL or fails if the http server isn't provided.
func (m *DiscoveryServerImpl) ServeDirectoryTD(
	serviceName string, tddJSON string) (tddURL string, err error) {
	// map of endpoints by scheme (wss, sse, ...)

	if m.httpServer == nil {
		return "", fmt.Errorf("ServeDirectoryTD: missing http server")
	}
	if serviceName == "" {
		serviceName = m.serviceName
	}
	publicRoute := m.httpServer.GetPublicRoute()
	// TBD: support for base path?
	wellKnownPath := directory.WellKnownWoTPath

	publicRoute.Get(wellKnownPath, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/td+json")
		_, _ = w.Write([]byte(tddJSON))
	})
	tddURL, err = url.JoinPath(m.httpServer.GetConnectURL(), wellKnownPath)

	// FIXME: server multiples?
	dnsSrv, err := ServeWotDiscovery(serviceName, tddURL, true, m.endpoints)

	if err != nil {
		slog.Error("Failed starting introduction server for DNS-SD",
			"TDD URL", tddURL,
			"err", err.Error())
		return tddURL, err
	}
	m.dnssdServers[serviceName] = dnsSrv
	return tddURL, nil
}

// ServeThingTD registers the given thing TD with the HTTP server and publishes
// its provisioning endpoint using DNS-SD discovery.
// Indended for use by Things that run servers.
//
// Since a device can publish multiple services, the serviceName is used
// in the discovery path. .wellknown/wot/{serviceName}
//
//	serviceName is the instance name to publish under. "" to use the TD ID
//	tdJSON is the Thing's TD in JSON format
func (m *DiscoveryServerImpl) ServeThingTD(
	serviceName string, tdJSON string) (err error) {

	slog.Info("DiscoveryServer. Serving Thing TD")

	tdDoc, err := td.UnmarshalTD(tdJSON)
	if err != nil {
		return fmt.Errorf("ServeThingTD: Invalid TD: %w", err)
	}
	httpPath := directory.WellKnownWoTPath
	if serviceName == "" {
		serviceName = tdDoc.ID
		httpPath = httpPath + "/" + serviceName
	}

	// serve the TD on the well-known http endpoint
	publicRoute := m.httpServer.GetPublicRoute()
	publicRoute.Get(httpPath, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(tdJSON))
	})

	// publish a discovery record
	thingTDURL, _ := url.JoinPath(m.httpServer.GetConnectURL(), httpPath)
	dnsSrv, err := ServeWotDiscovery(serviceName, thingTDURL, false, nil)
	if err != nil {
		slog.Error("Failed starting introduction server for DNS-SD",
			"Thing TD URL", thingTDURL,
			"err", err.Error())
		return err
	}
	m.dnssdServers[serviceName] = dnsSrv
	return nil
}

// Stop any running services and release resources
func (m *DiscoveryServerImpl) Stop() {
	m.mux.Lock()
	defer m.mux.Unlock()
	slog.Info("Stop: Stopping discovery transport servers", "count", len(m.dnssdServers))
	if m.dnssdServers != nil {
		for _, dnsSrv := range m.dnssdServers {
			dnsSrv.Shutdown()
		}
		m.dnssdServers = nil
		// the DNS server takes a wee bit of time to really stop
		// Wait this wee bit to prevent a race running tests
		time.Sleep(time.Millisecond)
	}
}

// NewDiscoveryServerImpl returns a ready-to-use discovery server instance.
//
// Call ServeThingTD or ServeDirectoryTD to serve a DNS-SD record.
//
// The thingID is set to the cell type. Note that the ID in the TDD might differ.
//
// When used in a cell chain together with a directory, this service must be placed
// after the directory in the chain, so it can find the directory to get its TDD,
// and any prevent it from intercepting a CreateThing request send by services.
//
//	serviceName is the default name under which to serve the discovery record.
//	httpServer is the server that serves the TD on the well-known endpoint.
//	tddJSON is the optional directory TDD as JSON to serve.
//	transports for TD security scheme, base URL and forms. Optional.
func NewDiscoveryServerImpl(serviceName string,
	httpServer api.IHttpServer,
	tddJSON string,
	endpoints map[string]string) (*DiscoveryServerImpl, error) {
	var err error

	// thingID is defined in the TDD and should match HiveCell thingID
	thingID := discovery.DiscoveryServerCellType

	srv := &DiscoveryServerImpl{
		HiveCellBase: cells.NewHiveCellBase(thingID, 0),
		serviceName:  serviceName,
		tddJSON:      tddJSON,
		endpoints:    endpoints,
		httpServer:   httpServer,
		dnssdServers: make(map[string]*zeroconf.Server),
	}

	if tddJSON != "" {
		slog.Info("Start: Starting discovery server - serving directory TD")
		_, err = srv.ServeDirectoryTD(serviceName, tddJSON)
	} else {
		slog.Info("Start: Starting discovery server - no TD served yet")
	}

	var _ discovery.IDiscoveryServer = srv // interface check
	return srv, err
}

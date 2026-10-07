package internal

import (
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/grandcat/zeroconf"
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/cells"
	"github.com/hiveot/hivekit/go/cells/directory"
	"github.com/hiveot/hivekit/go/cells/discovery"
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

	// hook to add forms to the given TD
	addFormsHook func(tdoc *td.TD)

	// The optional directory TD to serve on start
	tdd *td.TD

	// optional additional endpoints to publish in the discovery record in addition to
	// the well-known exploration URL.
	endpoints map[string]string

	// service discovery using mDNS per thingID
	dnssdServers map[string]*zeroconf.Server

	// the http server that servers the exploration endpoint.
	httpServer api.IHttpServer

	mux sync.RWMutex
}

// Handle request to serve a directory or Thing TD.
// Intended for use in a cell chain where a device or directory publishes its TD for discovery.
func (srv *DiscoveryServerImpl) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {

	// no need to check the discovery thingID, the action name in this chain is sufficient.
	if req.Operation == td.OpInvokeAction {
		switch req.Name {
		case discovery.ServeDirectoryTDAction:
			tdoc, _ := td.UnmarshalTD(req.ToString(0))
			tddURL, err := srv.ServeDirectoryTD(tdoc.ID, tdoc)
			resp := req.CreateResponse(tddURL, err)
			return replyTo(resp)

		case discovery.ServeThingTDAction:
			tdoc, _ := td.UnmarshalTD(req.ToString(0))
			tdURL, err := srv.ServeThingTD(tdoc.ID, tdoc)
			resp := req.CreateResponse(tdURL, err)
			return replyTo(resp)

		case directory.UpdateThingAction, directory.CreateThingAction:
			// When a device or service publishes their TD it is send as a create thing
			// request.
			// With a discovery server in the chain this is used to publish a Thing
			// discovery record. If the chain also contains a directory then the directory
			// MUST be placed before the discovery service to avoid it intercepting of the
			// request.
			tdoc, _ := td.UnmarshalTD(req.ToString(0))
			// add forms if missing
			if tdoc.Base == "" && len(tdoc.Forms) == 0 {
				if srv.addFormsHook != nil {
					srv.addFormsHook(tdoc)
				}
			}
			// this is a Thing, so an instance name is needed to differentiate
			// between things.
			instanceName := tdoc.ID
			tdURL, err := srv.ServeThingTD(instanceName, tdoc)
			resp := req.CreateResponse(tdURL, err)
			return replyTo(resp)
		}
	}
	return srv.ForwardRequest(req, replyTo)
}

// ServeTD serves the TD using DNS-SD.
//
// This registers the download URL with the configured http server on the
// 'well-known' endpoint included in the discovery record.
//
// An instanceName of "" results in using the WoT well-known download path and a
// service name of hostname:thingID
//
// If multiple records should be served then provide a instanceName. It will be added to
// the download path, eg: http://.well-known/wot/{instanceName} and to the
// service record instance as "{hostname}:{instanceName}"
//
// Users must call Release on the zeroconf DNS server when done.
//
//	instanceName is the instance name to publish the TD with.
//	discoType is one of DISCO_TYPE_DIRECTORY|GATEWAY|THING
//	tdoc is the Thing's TD to serve
//	isGateway flag, this device is a gateway
//
// This returns the TD download URL or an error
func (srv *DiscoveryServerImpl) ServeTD(
	instanceName string, discoType string, tdoc *td.TD) (tdURL string, err error) {

	var httpPath string
	hostName, _ := os.Hostname()

	if tdoc == nil {
		return "", errors.New("Missing tdoc argument")
	}

	if instanceName == "" {
		httpPath = discovery.WellKnownHttpPath
		instanceName = hostName + ":" + tdoc.ID
	} else {
		httpPath = discovery.WellKnownHttpPath + "/" + instanceName
	}
	tdJSON := tdoc.ToJSON()

	// serve the TD on the well-known http endpoint
	publicRoute := srv.httpServer.GetPublicRoute()
	publicRoute.Get(httpPath, func(w http.ResponseWriter, r *http.Request) {
		_ = tdoc
		_ = tdJSON
		_, _ = w.Write([]byte(tdJSON))
	})

	// publish a discovery record
	tdURL, _ = url.JoinPath(srv.httpServer.GetConnectURL(), httpPath)
	dnsSrv, err := ServeWotDiscovery(instanceName, tdURL, discoType, nil)
	if err != nil {
		slog.Error("ServeTD: Failed starting introduction server for DNS-SD",
			"tdURL", tdURL,
			"err", err.Error())
		return tdURL, err
	}
	srv.dnssdServers[instanceName] = dnsSrv
	return tdURL, nil
}

// ServeDirectoryTD registers the given directory TD with the http server
// and publishes its endpoint using DNS-SD discovery.
//
// This can be invoked directly of via a ServeDirectoryTDAction request.
//
//	serviceName is the DNS record name, required for multiple instances. Use "" for hostname.
//	tddJSON must be provided by a directory that implements the affordances.
//
// If a list of transports is available this updates the TD security scheme,
// base URL and forms.
//
// This aims to be compliant with https://w3c.github.io/wot-discovery/#exploration-server
//
// This returns the TD download URL or an error
func (srv *DiscoveryServerImpl) ServeDirectoryTD(
	serviceName string, tdoc *td.TD) (tdURL string, err error) {
	if tdoc == nil {
		return "", errors.New("Missing TD")
	}
	slog.Info("ServeDirectoryTD. Serving Directory TD",
		"serviceName", serviceName, "thingID", tdoc.ID)
	return srv.ServeTD(serviceName, discovery.DISCO_TYPE_DIRECTORY, tdoc)

}

// ServeGatewayTD registers the given thing TD as a gateway.
//
// This sets the ThingType to DISCO_TYPE_GATEWAY, which identifies
// the device as a gateway. This is not a WoT recognized discovery type
// as WoT does not support gateways.
//
// Since a device can publish multiple services, the serviceName is used
// in the discovery path. .wellknown/wot/{serviceName}
//
//	serviceName is the DNS record name, required for multiple instances. Use "" for hostname.
//	tdoc is the Thing's TD
//
// This returns the TD download URL or an error
func (srv *DiscoveryServerImpl) ServeGatewayTD(
	serviceName string, tdoc *td.TD) (tdURL string, err error) {

	// serviceName = tdoc.ID

	if tdoc == nil {
		return "", errors.New("Missing TD")
	}
	slog.Info("ServeGatewayTD. Serving Gateway TD",
		"serviceName", serviceName, "thingID", tdoc.ID)

	return srv.ServeTD(serviceName, discovery.DISCO_TYPE_GATEWAY, tdoc)
}

// ServeThingTD registers the given thing TD with the HTTP server and publishes
// its provisioning endpoint using DNS-SD discovery.
// Indended for use by Things that run servers.
//
// The default provisioning endpoint is the well-known discovery path
// "/.well-known/wot". If a serviceName is provided then this is added to the
// path in order to support multiple devices.
//
//	instanceName is the DNS record name, required for multiple instances.
//	  Use "" for serving the TD on the default .well-known discovery path.
//	tdoc is the Thing's TD to serve
//
// This returns the TD download URL or an error
func (srv *DiscoveryServerImpl) ServeThingTD(
	instanceName string, tdoc *td.TD) (tdURL string, err error) {

	if tdoc == nil {
		return "", errors.New("Missing TD")
	}
	slog.Info("ServeThingTD. Serving Thing TD",
		"serviceName", instanceName, "thingID", tdoc.ID)
	return srv.ServeTD(instanceName, discovery.DISCO_TYPE_THING, tdoc)
}

// Stop any running services and release resources
func (srv *DiscoveryServerImpl) Stop() {
	srv.mux.Lock()
	defer srv.mux.Unlock()
	slog.Info("Stop: Stopping discovery transport servers", "count", len(srv.dnssdServers))
	if srv.dnssdServers != nil {
		for _, dnsSrv := range srv.dnssdServers {
			dnsSrv.Shutdown()
		}
		srv.dnssdServers = nil
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
//	httpServer is the server that serves the TD on the well-known endpoint.
//	tdd is the optional directory TDD to serve.
//	transports for TD security scheme, base URL and forms. Optional.
//	addForms is an optional hook to add missing forms to published TDs
func NewDiscoveryServerImpl(
	httpServer api.IHttpServer, tdd *td.TD,
	endpoints map[string]string,
	addForms func(*td.TD)) (*DiscoveryServerImpl, error) {
	var err error

	// thingID is defined in the TDD and should match HiveCell thingID
	thingID := discovery.DiscoveryServerDefaultThingID

	srv := &DiscoveryServerImpl{
		HiveCellBase: cells.NewHiveCellBase(thingID),
		addFormsHook: addForms,
		dnssdServers: make(map[string]*zeroconf.Server),
		endpoints:    endpoints,
		httpServer:   httpServer,
		tdd:          tdd,
	}
	if tdd != nil {
		_, err = srv.ServeDirectoryTD(tdd.ID, tdd)
	} else {
		slog.Info("Start: Starting discovery server - no TD served yet")
	}

	var _ discovery.IDiscoveryServer = srv // interface check
	return srv, err
}

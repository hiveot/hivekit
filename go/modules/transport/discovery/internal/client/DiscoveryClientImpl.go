package internal

import (
	"crypto/x509"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/grandcat/zeroconf"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/factory"
	"github.com/hiveot/hivekit/go/modules/transport/discovery"
	tlsclientpkg "github.com/hiveot/hivekit/go/modules/transport/tlsclient/pkg"
)

// Client for discovery of WoT devices and directories
//
// When launched through the module factory this auto-discovers a directory TDD and
// a gateway TD (if available) on Start.
type DiscoveryClientImpl struct {
	*modules.HiveModuleBase

	// ca certificate
	caCert *x509.Certificate

	// optional update the discovery results in the app environment
	env *factory.AppEnvironment

	// discovery directory info when running DiscoveryFirstDirectory
	dirURL string // the directory TD instance
	// the discovered directory TD if available
	dirTD *td.TD

	// auto run discovery on startup
	discoverOnStart bool

	// discovery of the server using the env serverURL
	serverURL string
	// the discovered server TD if available
	serverTD *td.TD

	// mux for access to discovered data
	mux sync.RWMutex
}

// discoverDirectories invokes a callback on each directory discovered
// The callback can return true to stop the process.
func (cl *DiscoveryClientImpl) DiscoverDirectories(instanceName string, maxWaitTime time.Duration,
	cb func(*discovery.DiscoveryResult) bool) ([]*discovery.DiscoveryResult, error) {

	drList := make([]*discovery.DiscoveryResult, 0)

	// run the scan to collect results
	_, err := DnsSDScan(instanceName, discovery.WOT_DIRECTORY_SERVICE_TYPE, maxWaitTime,
		func(rec *zeroconf.ServiceEntry) bool {
			var stop = false
			// create a discovery record for the service entry
			discoRecord := cl.ParseZeroconfServiceEntry(rec)
			drList = append(drList, discoRecord)
			if cb != nil {
				stop = cb(discoRecord)
			}
			return stop
		})
	return drList, err
}

// DiscoverDirectory returns the first discovered record for a directory
//
// This returns nil with no error if discovery ran successful but no record was found.
func (cl *DiscoveryClientImpl) DiscoverFirstDirectory(
	instanceName string, searchTime time.Duration) (rec0 *discovery.DiscoveryResult, err error) {

	// stop on the first result
	records, err := cl.DiscoverDirectories(instanceName, searchTime,
		func(*discovery.DiscoveryResult) bool { return true })

	if len(records) == 0 {
		return nil, err
	}

	// Determine the directory URL and download the TD.
	rec0 = records[0]
	return rec0, nil
}

// DiscoverFirstGateway returns the discovery record if the first gateway server.
func (cl *DiscoveryClientImpl) DiscoverFirstGateway(
	instanceName string, searchTime time.Duration) (rec0 *discovery.DiscoveryResult, err error) {
	if instanceName == "" {
		instanceName = discovery.HIVEOT_GATEWAY_SERVICE_TYPE
	}
	if searchTime == 0 {
		searchTime = time.Second * 3
	}

	// return on the first result
	records, err := cl.DiscoverThings(
		instanceName, searchTime, func(res *discovery.DiscoveryResult) bool {
			return true
		})
	if len(records) == 0 {
		return nil, err
	}
	rec0 = records[0]
	return rec0, nil
}

// DiscoverThings returns discovery records of wot Things that publish themselves on the network.
//
// Intended for environments where things run servers themselves (instead of using a hub/gateway).
//
//	instanceName is optional and intended to search for a particular instance by name, such as 'hub'.
//	duration is the time to search for.
//	cb is the callback to invoke when a match is found. Returns true to stop.
//
// This returns a list of all discoveries
func (cl *DiscoveryClientImpl) DiscoverThings(
	instanceName string, maxWaitTime time.Duration,
	cb func(*discovery.DiscoveryResult) bool) ([]*discovery.DiscoveryResult, error) {

	var mux sync.RWMutex

	drList := make([]*discovery.DiscoveryResult, 0)

	// run the scan to collect results
	_, err := DnsSDScan(instanceName, discovery.WOT_THING_SERVICE_TYPE, maxWaitTime,
		func(rec *zeroconf.ServiceEntry) bool {

			// create a discovery record for the service entry
			discoRecord := cl.ParseZeroconfServiceEntry(rec)
			mux.Lock()
			drList = append(drList, discoRecord)
			mux.Unlock()
			if cb != nil {
				return cb(discoRecord) // return true to stop
			}
			return false // keep looking
		})

	mux.Lock()
	result := drList
	mux.Unlock()
	return result, err
}

// Discover a TDD and return the result or an error
//
// If a directory URL is known then load the TDD from the URL, otherwise do a DNS-SD search
// for the directory to get the URL.
//
// This updates this client's directory TD and returns the TDD and its loaded JSON.
// If no TDD is found this responds with an error.
//
// If multiple requests are send then each will update the directory TDD if found.
// If a directory URL
func (cl *DiscoveryClientImpl) DiscoverTDD() (tdd *td.TD, tddJson string, err error) {

	if cl.dirURL == "" {
		rec0, _ := cl.DiscoverFirstDirectory("", time.Second)
		cl.dirURL = rec0.AsURL()
	}
	if cl.dirURL == "" {
		err = fmt.Errorf("No directory was discovered")
	} else {
		cl.dirTD, tddJson, err = cl.LoadTD(cl.dirURL, cl.caCert)
	}
	return cl.dirTD, tddJson, err
}

// Handle requests to discover directory TD.
func (cl *DiscoveryClientImpl) HandleRequest(
	req *msg.RequestMessage, replyTo msg.ResponseHandler) error {

	if req.Operation == td.OpInvokeAction && req.Name == discovery.DiscoverDirectoryAction {
		_, tddJson, err := cl.DiscoverTDD()
		resp := req.CreateResponse(tddJson, err)
		return replyTo(resp)
	}
	return cl.ForwardRequest(req, replyTo)
}

// LoadTD a TD document from a discovery result.
//
// Intended for discovery of a thing or directory TD. This downloads the TD Json using
// the URL in the discovery record.
//
// rec points to the discovery record.
// caCert is optional CA to verify the server validity. nil skips this validation.
//
// This returns the TD, its JSON or an error if none is found
func (cl *DiscoveryClientImpl) LoadTD(tdURL string, caCert *x509.Certificate) (tdoc *td.TD, tdJSON string, err error) {

	slog.Info("DownloadTD", "url", tdURL)
	parts, err := url.Parse(tdURL)
	if err != nil {
		return nil, "", err
	}
	if strings.ToLower(parts.Scheme) != "https" {
		return nil, "", fmt.Errorf("Unknown scheme '%s', only http is supported", parts.Scheme)
	}
	httpCl := tlsclientpkg.NewTLSClient(parts.Host, caCert, 0)
	resp, statusCode, err := httpCl.Get(parts.Path)
	_ = statusCode
	if err != nil {
		return nil, "", fmt.Errorf("DownloadTD: download failed: %w", err)
	}
	tdJSON = string(resp)
	tdDoc, err := td.UnmarshalTD(tdJSON)
	return tdDoc, tdJSON, err
}

// Return the gateway server URL discovered during start
//
// If an app environment was provided on start and it contained a server URL then
// it will return this server URL instead.
// func (cl *DiscoveryClient) GetServerURL() string {
// 	cl.mux.RLock()
// 	defer cl.mux.RUnlock()

// 	return cl.serverURL
// }

// Return the directory connection TD instance discovered during Start
func (cl *DiscoveryClientImpl) GetTDD() (dirTD *td.TD) {
	cl.mux.RLock()
	defer cl.mux.RUnlock()

	return cl.dirTD
}

// Return the server TD instance discovered during Start
// // This returns nil if no server was discovered.
func (cl *DiscoveryClientImpl) GetServerTD() *td.TD {
	cl.mux.RLock()
	defer cl.mux.RUnlock()

	return cl.serverTD
}

// Convert a zeroconf result to a hiveot discovery record
func (cl *DiscoveryClientImpl) ParseZeroconfServiceEntry(
	rec *zeroconf.ServiceEntry) *discovery.DiscoveryResult {

	discoResult := discovery.DiscoveryResult{
		Params:   make(map[string]string),
		Instance: rec.Instance,
		Port:     rec.Port,
	}

	// determine the address string
	// use the local IP if provided
	if len(rec.AddrIPv4) > 0 {
		discoResult.Addr = rec.AddrIPv4[0].String()
	} else if len(rec.AddrIPv6) > 0 {
		discoResult.Addr = rec.AddrIPv6[0].String()
	} else {
		// fall back to use host.domainname
		discoResult.Addr = rec.HostName
	}

	// https://w3c.github.io/wot-discovery/#introduction-dns-sd-sec
	if rec.Service == discovery.WOT_DIRECTORY_SERVICE_TYPE {
		discoResult.IsDirectory = true
	} else if rec.Service == discovery.WOT_THING_SERVICE_TYPE {
		discoResult.IsThing = true
	} else {
		// not sure what this is
	}

	// For TCP-based services, the following information MUST be included in the
	// TXT record that is pointed to by the Service Instance Name:
	for _, txtRecord := range rec.Text {
		kv := strings.Split(txtRecord, "=")
		if len(kv) != 2 {
			slog.Info("DiscoverService: Ignoring non key-value in TXT record", "key", txtRecord)
			continue
		}
		key := kv[0]
		val := kv[1]
		if key == "td" {
			discoResult.TD = val // Absolute pathname of the TD/TDD
		} else if key == "type" {
			discoResult.Type = val // Type of TD, "Thing" or "Directory" or "Hiveot"
			discoResult.IsDirectory = val == "Directory"
			discoResult.IsThing = val == "Thing"
		} else if key == "scheme" {
			// http (default), https, coap+tcp, coaps+tcp
			discoResult.Schema = val // Scheme part of URL
		} else if key == discovery.WSSEndpoint {
			// 'base' is specific to hiveot to provide a default connection URL
			discoResult.WSSEndpoint = val
		} else if key == discovery.SSEEndpoint {
			// 'base' is specific to hiveot to provide a default connection URL
			discoResult.SSEEndpoint = val
		} else if key == discovery.AuthEndpoint {
			discoResult.AuthEndpoint = val
		}
		discoResult.Params[key] = val
	}
	return &discoResult
}

// Start runs a discovery of directory and gateway.
//
// If an application environment is provided it will also update the directory URL
// if it isn't set manually.
func (cl *DiscoveryClientImpl) Start() (err error) {
	var rec0 *discovery.DiscoveryResult

	// first obtain the directory URL
	if cl.dirURL == "" && cl.discoverOnStart {
		rec0, err = cl.DiscoverFirstDirectory("", 0)
		if rec0 != nil {
			cl.dirURL = rec0.AsURL()
		}
		if cl.dirURL == "" {
			slog.Warn("Start: No directories are discovered on the local network.")
		}
	}
	// next, use it to load the directory TD
	if cl.dirURL != "" {
		cl.dirTD, _, err = cl.LoadTD(cl.dirURL, cl.caCert)

		if err != nil {
			slog.Warn("Start: Directory is not available ..", "err", err.Error())
			// "directoryURL", dirURL, "err", err.Error())
		}

		// optionally update the factory environment to share the results with other modules
		if cl.env != nil && cl.env.DirectoryURL == "" {
			cl.env.DirectoryURL = cl.dirURL
		}
	}

	// FIXME: determine the server URL
	// what is a serverURL anyways? does this work without forms?
	//   RC doesn't need forms, so no need for a TD?...
	// what about connecting to a device?
	//   do devices expose their TD?
	//
	// option1: serverURL is hiveot transport endpoint. Just connect and send requests
	// option2: don't use serverURL. this is discovery so discover the URL
	if cl.serverURL == "" && cl.discoverOnStart {
		rec0, err := cl.DiscoverFirstGateway("", time.Second)
		if err != nil {
			cl.serverURL = rec0.AsURL()
		}

		if cl.serverURL != "" {
			cl.serverTD, _, err = cl.LoadTD(cl.serverURL, cl.caCert)
		}
		// optionally update the factory environment to share the results with other modules
		if cl.env != nil && cl.env.ServerURL == "" {
			cl.env.ServerURL = cl.serverURL
		}
	}

	return nil
}

// NewDiscoveryClientImpl creates a new instance of a discovery client
//
// appEnv is optional. On Start it will be updated with the discovered directory and server.
func NewDiscoveryClientImpl(appEnv *factory.AppEnvironment, discoOnStart bool) *DiscoveryClientImpl {
	cl := &DiscoveryClientImpl{
		HiveModuleBase:  modules.NewHiveModuleBase("", 0),
		env:             appEnv,
		discoverOnStart: discoOnStart,
	}
	if appEnv != nil {
		cl.caCert = appEnv.CaCert
		cl.dirURL = appEnv.DirectoryURL
		cl.serverURL = appEnv.ServerURL
	}
	return cl
}

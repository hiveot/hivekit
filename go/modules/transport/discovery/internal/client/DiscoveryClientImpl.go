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
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules"
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
	env *api.AppEnvironment

	// auto run discovery on startup
	discoverOnStart bool

	// mux for access to discovered data
	mux sync.RWMutex
}

func (cl *DiscoveryClientImpl) _discover(
	instanceName string, serviceType string, maxWaitTime time.Duration,
	cb func(*discovery.DiscoveryResult) bool) ([]*discovery.DiscoveryResult, error) {

	mux := sync.RWMutex{}
	drList := make([]*discovery.DiscoveryResult, 0)

	// run the scan to collect results
	_, err := DnsSDScan(instanceName, serviceType, maxWaitTime,
		func(rec *zeroconf.ServiceEntry) bool {
			var stop = false
			// create a discovery record for the service entry
			discoRecord := cl.ParseZeroconfServiceEntry(rec)
			// when maxWaitTime is reached there can be a race with this callback
			mux.Lock()
			drList = append(drList, discoRecord)
			mux.Unlock()
			if cb != nil {
				stop = cb(discoRecord)
			}
			return stop
		})
	mux.RLock()
	result := drList
	mux.RUnlock()
	return result, err
}

// discoverDirectories invokes a callback on each directory discovered
// The callback can return true to stop the process.
func (cl *DiscoveryClientImpl) DiscoverDirectories(maxWaitTime time.Duration,
	cb func(*discovery.DiscoveryResult) bool) ([]*discovery.DiscoveryResult, error) {

	recs, err := cl._discover("", discovery.WOT_DIRECTORY_SERVICE_TYPE, maxWaitTime, cb)
	return recs, err
}

// Discover all directories on the local network and return their TDs.
// If the TD cannot be downloaded then it is ignored in the result.
func (cl *DiscoveryClientImpl) DiscoverDirectoryTDs(
	searchTime time.Duration) (recs []*discovery.DiscoveryResult, tddList []*td.TD) {

	recs, _ = cl.DiscoverDirectories(searchTime, nil)
	tddList = make([]*td.TD, 0, len(recs))

	for _, rec := range recs {
		dirURL := rec.AsURL()
		if dirURL != "" {
			dirTD, _, err := cl.LoadTD(dirURL)
			if err == nil {
				tddList = append(tddList, dirTD)
			}
		}
	}
	return recs, tddList
}

// DiscoverDirectory returns the first discovered record for a directory
//
// This returns nil with no error if discovery ran successful but no record was found.
func (cl *DiscoveryClientImpl) DiscoverFirstDirectory(
	instanceName string, maxWaitTime time.Duration) (rec0 *discovery.DiscoveryResult, err error) {

	records, err := cl._discover(instanceName, discovery.WOT_DIRECTORY_SERVICE_TYPE, maxWaitTime,
		// stop on the first result
		func(*discovery.DiscoveryResult) bool { return true })

	if len(records) == 0 {
		return nil, fmt.Errorf("DiscoverFirstDirectory: No directory was found")
	}

	// Determine the directory URL and download the TD.
	rec0 = records[0]
	return rec0, nil
}

// Discover the first directory TD and return the result or an error
//
// If a directory URL is known then load the TDD from the URL, otherwise do a DNS-SD search
// for the directory to get the URL.
//
// This updates this client's directory TD and returns the TDD and its loaded JSON.
// If no TDD is found this responds with an error.
//
// If multiple requests are send then each will update the directory TDD if found.
// If a directory URL
func (cl *DiscoveryClientImpl) DiscoverFirstDirectoryTD(
	instanceName string, maxWaitTime time.Duration) (dirTD *td.TD, tddJson string, err error) {
	var rec0 *discovery.DiscoveryResult

	var dirURL string
	rec0, err = cl.DiscoverFirstDirectory(instanceName, maxWaitTime)

	if rec0 != nil {
		dirURL = rec0.AsURL()
	}
	if dirURL == "" {
		err = fmt.Errorf("No directory was discovered")
	} else {
		dirTD, tddJson, err = cl.LoadTD(dirURL)
	}
	return dirTD, tddJson, err
}

// DiscoverFirstGateway returns the discovery record if the first gateway server.
// func (cl *DiscoveryClientImpl) DiscoverFirstGateway(
// 	instanceName string, searchTime time.Duration) (rec0 *discovery.DiscoveryResult, err error) {
// 	if instanceName == "" {
// 		instanceName = discovery.HIVEOT_GATEWAY_SERVICE_TYPE
// 	}
// 	if searchTime == 0 {
// 		searchTime = time.Second * 3
// 	}

// 	// return on the first result
// 	records, err := cl.DiscoverThings(
// 		instanceName, searchTime, func(res *discovery.DiscoveryResult) bool {
// 			return true
// 		})
// 	if len(records) == 0 {
// 		return nil, err
// 	}
// 	rec0 = records[0]
// 	return rec0, nil
// }

// DiscoverThings returns discovery records of all wot Things that publish themselves on the network.
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

	records, err := cl._discover(instanceName, discovery.WOT_THING_SERVICE_TYPE, maxWaitTime, cb)
	result := records
	return result, err
}

// Discover things and download their TD
func (cl *DiscoveryClientImpl) DiscoverThingTDs(
	instanceName string, maxWaitTime time.Duration, cb func(*td.TD) bool) ([]*discovery.DiscoveryResult, []*td.TD) {

	tddList := make([]*td.TD, 0)
	recs, _ := cl.DiscoverThings(instanceName, maxWaitTime, func(rec *discovery.DiscoveryResult) bool {
		stop := false
		dirURL := rec.AsURL()
		if dirURL != "" {
			dirTD, _, err := cl.LoadTD(dirURL)
			if err == nil {
				tddList = append(tddList, dirTD)
				if cb != nil {
					stop = cb(dirTD)
				}
			}
		}
		return stop
	})

	return recs, tddList
}

// Handle requests to discover directory TD.
func (cl *DiscoveryClientImpl) HandleRequest(
	req *msg.RequestMessage, replyTo msg.ResponseHandler) error {

	if req.Operation == td.OpInvokeAction && req.Name == discovery.DiscoverDirectoryAction {
		_, tddJson, err := cl.DiscoverFirstDirectoryTD("", 0)
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
//
// This returns the TD, its JSON or an error if none is found
func (cl *DiscoveryClientImpl) LoadTD(tdURL string) (tdoc *td.TD, tdJSON string, err error) {

	slog.Info("DownloadTD", "url", tdURL)
	parts, err := url.Parse(tdURL)
	if err != nil {
		return nil, "", err
	}
	if strings.ToLower(parts.Scheme) != "https" {
		return nil, "", fmt.Errorf("Unknown scheme '%s', only http is supported", parts.Scheme)
	}
	httpCl := tlsclientpkg.NewTLSClient(parts.Host, cl.caCert, 0)
	resp, statusCode, err := httpCl.Get(parts.Path)
	_ = statusCode
	if err != nil {
		return nil, "", fmt.Errorf("DownloadTD: download failed: %w", err)
	}
	tdJSON = string(resp)
	tdDoc, err := td.UnmarshalTD(tdJSON)
	if err != nil {
		err = fmt.Errorf("LoadTD: TD loaded from '%s' but it doesn't appear to be valid json: %w",
			parts.Host+"/"+parts.Path, err)
	}
	return tdDoc, tdJSON, err
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

// Start the discovery client.
//
// If an application environment is provided and no directory URL is set,
// then run a discovery to update the AppEnvironment directory URL and
// Server URL. (if empty)
func (cl *DiscoveryClientImpl) Start() (err error) {

	var rec0 *discovery.DiscoveryResult
	var dirURL string
	var tddURL string

	// discover to populate the app env if needed
	if cl.discoverOnStart && cl.env != nil && cl.env.DirectoryURL == "" {

		// first obtain the directory exploration URL for downloading a TDD
		rec0, err = cl.DiscoverFirstDirectory("", 0)
		if rec0 != nil {
			tddURL = rec0.AsURL()
		}
		if tddURL == "" {
			slog.Warn("Start: No directories are discovered on the local network.")
			return nil
		}

		// next, use it to load the directory TD from the exploration endpoint
		dirTD, _, err := cl.LoadTD(tddURL)

		if err != nil {
			slog.Warn("Start: Directory is not available at the discovered URL",
				"tddURL", tddURL,
				"err", err.Error())
			return nil // not fatal
			// "directoryURL", dirURL, "err", err.Error())
		} else {
			// validate the URL
			parts, err := url.Parse(dirTD.Base)
			_ = parts
			if err != nil || parts.Host == "" {
				slog.Warn("Start: Directory found but its Base is not a valid URL",
					"Base", dirTD.Base)
			} else {
				// Base is a valid URL. Use it as the directory connection url.
				// Technically, each action request can have a different URL for the directory
				// subscription and each of the actions. Ignore this for now and use the TDD Base
				// as the directory URL.
				dirURL = dirTD.Base
				slog.Info("Start: Directory found", "URL", dirURL)
			}
		}

		// update the factory environment to share the results with other modules
		if dirURL != "" {
			if cl.env.DirectoryURL == "" {
				cl.env.DirectoryURL = dirURL
			}
			// in case a gateway server is used the gateway server URL is the same as that of the directory.
			if cl.env.ServerURL == "" {
				cl.env.ServerURL = dirTD.Base
			}
		}
		// if we have a valid Directory TDD then send a notification.
	}

	return nil
}

// NewDiscoveryClientImpl creates a new instance of a discovery client.
//
// If an appEnv is provided and its DirectoryURL is empty, and discoOnStart is enabled
// then Start will run in initial directory discovery and update appEnv with the
// resulting directory.
//
// If appEnv is provided and discovery on Start is successful then update appEnv with
// the discovered directory URL. The directory client can use this to connect to the directory.
func NewDiscoveryClientImpl(appEnv *api.AppEnvironment, discoOnStart bool) *DiscoveryClientImpl {
	cl := &DiscoveryClientImpl{
		HiveModuleBase:  modules.NewHiveModuleBase("", 0),
		env:             appEnv,
		discoverOnStart: discoOnStart,
	}
	if appEnv != nil {
		cl.caCert = appEnv.CaCert
	}
	var _ discovery.IDiscoveryClient = cl // interface check
	return cl
}

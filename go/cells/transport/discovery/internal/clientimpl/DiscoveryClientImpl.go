package clientimpl

import (
	"crypto/x509"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/grandcat/zeroconf"
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/cells"
	"github.com/hiveot/hivekit/go/cells/transport/discovery"
)

// Client for discovery of WoT devices and directories
//
// When launched through the cell factory this auto-discovers a directory TDD and
// a gateway TD (if available) on Start.
type DiscoveryClientImpl struct {
	*cells.HiveCellBase

	// ca certificates
	rootCAs *x509.CertPool

	// optional update the discovery results in the app environment
	env *api.HiveEnvironment

	// auto run discovery on startup
	discoverOnStart bool

	// mux for access to discovered data
	mux sync.RWMutex
}

// perform a DNS-SD discovery for WoT things, including directories
func (cl *DiscoveryClientImpl) _dnssd_discover(
	instanceName string, serviceType string, maxWaitTime time.Duration,
	cb func(*discovery.DiscoveryResult) bool) ([]*discovery.DiscoveryResult, error) {

	drList := make([]*discovery.DiscoveryResult, 0)

	// run the scan to collect results
	_, err := DnsSDScan(instanceName, serviceType, maxWaitTime,
		func(svcRec *zeroconf.ServiceEntry) bool {
			var stop = false
			// create a discovery record for the service entry
			discoRecord := cl.ParseZeroconfServiceEntry(svcRec)
			// when maxWaitTime is reached there can be a race with this callback
			drList = append(drList, discoRecord)
			if cb != nil {
				stop = cb(discoRecord)
			}
			return stop
		})
	result := drList[:]
	return result, err
}

// discoverDirectories invokes a callback on each discovered directory.
//
// The callback returns true to stop the process or false to continue.
// This is using the _wot._tcp service type, not _directory._sub._wot._tcp,
// as a directory record is identified by the "Type" field.
func (cl *DiscoveryClientImpl) DiscoverDirectories(maxWaitTime time.Duration,
	cb func(*discovery.DiscoveryResult) bool) ([]*discovery.DiscoveryResult, error) {

	// serviceType := discovery.WOT_SERVICE_TYPE + "," + discovery.WOT_DIRECTORY_SUB_TYPE
	// while a subtype offers filtering, the record determines whether
	// it is a directory....might as welll not use the subtype in discovery.
	serviceType := discovery.WOT_SERVICE_TYPE
	dirRecs := make([]*discovery.DiscoveryResult, 0)

	_, err := cl._dnssd_discover("", serviceType, maxWaitTime,
		func(rec *discovery.DiscoveryResult) bool {
			stop := false
			// filter on directories
			if strings.ToLower(rec.Type) != "directory" {
				return false
			}
			dirRecs = append(dirRecs, rec)
			if cb != nil {
				stop = cb(rec)
			}
			return stop
		})

	return dirRecs, err
}

// Discover all directories on the local network and return their TDs.
// If the TD cannot be downloaded then it is ignored in the result.
func (cl *DiscoveryClientImpl) DiscoverDirectoryTDs(
	searchTime time.Duration) (recs []*discovery.DiscoveryResult, tddList []*td.TD) {

	tddList = make([]*td.TD, 0, len(recs))
	recs, _ = cl.DiscoverDirectories(searchTime, nil)

	for _, rec := range recs {
		dirURL := rec.AsURL()
		if dirURL != "" {
			dirTD, _, err := LoadTD(dirURL, cl.rootCAs)
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
	instanceName string, maxWaitTime time.Duration) (first *discovery.DiscoveryResult, err error) {

	// stop on the first result
	_, err = cl.DiscoverDirectories(
		maxWaitTime, func(rec *discovery.DiscoveryResult) bool {
			if instanceName == "" || instanceName == rec.Instance {
				first = rec
				return true
			}
			return false
		})

	if first == nil {
		return nil, fmt.Errorf("DiscoverFirstDirectory: No directory was found")
	}

	return first, nil
}

// Discover the first directory TDD and return the result or an error
//
//	searchID optionally filters on a specific instance name or directory thingID
func (cl *DiscoveryClientImpl) DiscoverFirstDirectoryTD(
	searchID string, maxWaitTime time.Duration) (
	dirTD *td.TD, tddURL string, tddJSON string, err error) {

	// stop on the first matching result
	_, err = cl.DiscoverDirectories(maxWaitTime, func(rec *discovery.DiscoveryResult) bool {
		// keep looking until a matching TD is found
		tddURL = rec.AsURL()
		if tddURL == "" {
			return false
		}
		recTD, recTddJSON, err := LoadTD(tddURL, cl.rootCAs)
		if err != nil || recTD == nil {
			return false
		}
		if searchID != "" {
			if searchID == recTD.ID || searchID == rec.Instance {
				dirTD = recTD
				tddJSON = recTddJSON
				return true
			}
			// keep looking
			return false
		}
		// any TDD will do
		dirTD = recTD
		tddJSON = recTddJSON
		return true
	})

	if dirTD == nil {
		err = fmt.Errorf("No directory was discovered")
	} else {
		err = nil
	}
	return dirTD, tddURL, tddJSON, err
}

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

	records, err := cl._dnssd_discover(instanceName, discovery.WOT_SERVICE_TYPE, maxWaitTime, cb)
	result := records
	return result, err
}

// Discover all things and download their TD
// This separates directories from devices
// NOTE: If a TD cannot be read this includes nil in the result so the
// records table matches the TD table.
func (cl *DiscoveryClientImpl) DiscoverThingTDs(
	instanceName string, maxWaitTime time.Duration,
	cb func(*td.TD) bool) (
	dirRecs []*discovery.DiscoveryResult, dirTDs []*td.TD,
	deviceRecs []*discovery.DiscoveryResult, deviceTDs []*td.TD) {

	dirRecs = make([]*discovery.DiscoveryResult, 0)
	deviceRecs = make([]*discovery.DiscoveryResult, 0)

	dirTDs = make([]*td.TD, 0)
	deviceTDs = make([]*td.TD, 0)
	cl.DiscoverThings(instanceName, maxWaitTime, func(rec *discovery.DiscoveryResult) bool {
		stop := false
		if rec.IsDirectory {
			dirRecs = append(dirRecs, rec)
		} else {
			deviceRecs = append(deviceRecs, rec)
		}
		tdURL := rec.AsURL()
		var tdoc *td.TD
		if tdURL != "" {
			tdoc, _, _ = LoadTD(tdURL, cl.rootCAs)
		}
		if rec.IsDirectory {
			dirTDs = append(dirTDs, tdoc)
		} else {
			deviceTDs = append(deviceTDs, tdoc)
		}
		// invoke the callback if a TD was successfully loaded
		if tdoc != nil && cb != nil {
			stop = cb(tdoc)
		}
		return stop
	})

	return dirRecs, dirTDs, deviceRecs, deviceTDs
}

// // Handle requests to discover directory TD.
// func (cl *DiscoveryClientImpl) HandleRequest(
// 	req *msg.RequestMessage, replyTo msg.ResponseHandler) error {

// 	if req.Operation == td.OpInvokeAction && req.Name == discovery.DiscoverDirectoryAction {
// 		_, _, tddJson, err := cl.DiscoverFirstDirectoryTD("", 0)
// 		resp := req.CreateResponse(tddJson, err)
// 		return replyTo(resp)
// 	}
// 	return cl.ForwardRequest(req, replyTo)
// }

// LoadTD a TD document from a discovery result.
//
// Intended for discovery of a thing or directory TD. This downloads the TD Json using
// the URL in the discovery record.
//
// rec points to the discovery record.
//
// This returns the TD, its JSON or an error if none is found
func (cl *DiscoveryClientImpl) LoadTD(tdURL string) (tdoc *td.TD, tdJSON string, err error) {
	return LoadTD(tdURL, cl.rootCAs)
}

// Convert a zeroconf result to a hiveot discovery record
func (cl *DiscoveryClientImpl) ParseZeroconfServiceEntry(
	rec *zeroconf.ServiceEntry) *discovery.DiscoveryResult {

	discoResult := discovery.DiscoveryResult{
		Params:   make(map[string]string),
		Instance: rec.Instance,
		Hostname: rec.HostName,
		Port:     rec.Port,
		Service:  rec.Service,
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

	// default to Thing unless a TXT "Type" record is present
	discoResult.IsThing = true

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
			//https://w3c.github.io/wot-discovery/#exploration-td-type-thingdirectory
			discoResult.Type = val // Type of TD, "Thing" or "Directory" or "Hiveot"
			discoResult.IsDirectory = strings.ToLower(val) == "directory"
			discoResult.IsThing = strings.ToLower(val) == "thing"
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

// locateDirectoryToUse attempts to locate a directory and returns its TDD.
//
// If a TDD URL is offered then first try to load the TDD from that URL. If this
// fails then return an error.
//
// If no URL is offered then search for a directory and return its TDD and URL.
//
// If no TDD can be found then return nil
func (cl *DiscoveryClientImpl) locateDirectoryToUse(offeredURL string, maxWaitTime time.Duration) (*td.TD, string) {
	var tddURL string

	// if a URL is offered then work with it.
	if offeredURL != "" {
		dirTDD, _, err := LoadTD(tddURL, cl.rootCAs)
		if err != nil {
			slog.Warn("discoverDirectory: Directory is not available at the discovered URL",
				"tddURL", tddURL,
				"err", err.Error())
		}
		return dirTDD, offeredURL
	}
	// no directory URL offered so go find one that can be read.
	dirTDD, tddURL, _, _ := cl.DiscoverFirstDirectoryTD("", maxWaitTime)

	return dirTDD, tddURL
}

// NewDiscoveryClientImpl returns a ready-to-use discovery client.
//
// Call DiscoverThings or DiscoverDirectories to start the discovery process.
//
// If an appEnv is provided and its DirectoryURL is empty, and discoOnStart is enabled
// then Start will run in initial directory discovery and update appEnv with the
// resulting directory.
//
// If appEnv is provided and discovery on Start is successful then update appEnv with
// the discovered directory URL. The directory client can use this to connect to the directory.
func NewDiscoveryClientImpl(
	appEnv *api.HiveEnvironment, discoOnStart bool) (*DiscoveryClientImpl, error) {
	var err error

	cl := &DiscoveryClientImpl{
		HiveCellBase:    cells.NewHiveCellBase("", 0),
		env:             appEnv,
		discoverOnStart: discoOnStart,
	}
	if appEnv != nil {
		cl.rootCAs = appEnv.GetRootCAs()
	}

	// discover a directory for the app environment
	if cl.discoverOnStart && cl.env != nil && appEnv.DirTD == nil {
		dirTDD, tddURL := cl.locateDirectoryToUse(appEnv.TDDURL, time.Second)

		if dirTDD == nil {
			slog.Warn("NewDiscoveryClientImpl. Downloading the directory TDD failed",
				"tddURL", appEnv.TDDURL)
		} else {
			slog.Info("NewDiscoveryClientImpl. Directory TDD downloaded successfully",
				"tddURL", tddURL)
			appEnv.DirTD = dirTDD
			appEnv.TDDURL = tddURL
		}
	}

	var _ discovery.IDiscoveryClient = cl // interface check
	return cl, err
}

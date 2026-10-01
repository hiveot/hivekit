package internal

import (
	"crypto/x509"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/grandcat/zeroconf"
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/cells"
	"github.com/hiveot/hivekit/go/cells/discovery"
	"github.com/teris-io/shortid"
)

// Client for discovery of WoT device, service, gateway or directory TD.
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

// Return the first discovered TD that matches the searchID and type.
func (cl *DiscoveryClientImpl) DiscoverFirstTD(
	searchID string, thingType string, maxWaitTime time.Duration) (
	td *td.TD) {
	_, dirList, _, thingList := cl.DiscoverTDs(searchID, thingType, true, maxWaitTime, nil)

	combined := append(dirList, thingList...)
	if len(combined) == 0 {
		return nil
	}
	return combined[0]
}

// Return the discovery record of the first thing that matches the searchID and type.
func (cl *DiscoveryClientImpl) DiscoverFirstThing(
	instanceName string, thingType string, maxWaitTime time.Duration) *discovery.DiscoveryResult {
	recs, _ := cl.DiscoverThings(instanceName, thingType, true, maxWaitTime, nil)
	if len(recs) == 0 {
		return nil
	}
	return recs[0]
}

// DiscoverThings returns discovery records of WoT Things that publish themselves on the network.
//
//	instanceName is optional and intended to search for a particular instance by name, such as 'hub'.
//	thingType is optional TXT type record or "" for all WoT records
//	first, flag, stop on first matching result
//	maxWaitTime is maximum time to wait for search to complete
//	cb is the callback to invoke when a match is found. Returns true to stop.
//
// This returns a list of all discoveries
func (cl *DiscoveryClientImpl) DiscoverThings(
	instanceName string, thingType string, first bool, maxWaitTime time.Duration,
	cb func(*discovery.DiscoveryResult) bool) ([]*discovery.DiscoveryResult, error) {

	recs := make([]*discovery.DiscoveryResult, 0)

	thingTypeLower := strings.ToLower(thingType)
	_, err := cl._dnssd_discover(
		instanceName, discovery.WOT_SERVICE_TYPE, maxWaitTime,
		func(rec *discovery.DiscoveryResult) bool {
			stop := false
			// filter on thing Type (Thing, Directory or Gateway)
			if thingTypeLower != "" && strings.ToLower(rec.Type) != thingTypeLower {
				return false
			}
			recs = append(recs, rec)
			if cb != nil {
				stop = cb(rec)
			}
			return stop || first
		})
	return recs, err
}

// Discover things and download their TD.
//
// This separates directories from devices
// NOTE: If a TD cannot be read this includes nil in the TD results so the
// records table matches the TD table.
//
//	searchID optionally filters on a specific instance name or directory thingID
//	thingType, THING_TYPE_THING|DIRECTORY|GATEWAY
//	first stops on first valid result
//	maxWaitTime is maximum time to wait for search to complete
//
// This returns the matching directory, gateway and thing TDs
func (cl *DiscoveryClientImpl) DiscoverTDs(
	searchID string, thingType string, first bool, maxWaitTime time.Duration,
	cb func(*td.TD) bool) (
	dirRecs []*discovery.DiscoveryResult, dirTDs []*td.TD,
	deviceRecs []*discovery.DiscoveryResult, deviceTDs []*td.TD) {

	dirRecs = make([]*discovery.DiscoveryResult, 0)
	deviceRecs = make([]*discovery.DiscoveryResult, 0)

	dirTDs = make([]*td.TD, 0)
	deviceTDs = make([]*td.TD, 0)
	cl.DiscoverThings("", thingType, false, maxWaitTime,
		func(rec *discovery.DiscoveryResult) bool {
			stop := false
			if rec.IsDirectory {
				dirRecs = append(dirRecs, rec)
			} else {
				deviceRecs = append(deviceRecs, rec)
			}
			// ignore records without a valid TD URL
			tdURL := rec.AsURL()
			if tdURL == "" {
				return false
			}
			tdoc, tdJSON, err := LoadTD(tdURL, cl.rootCAs)
			_ = tdJSON
			if err != nil || tdoc == nil {
				return false
			}
			// filter on records that don't have the searchID in either instance name
			// or as the thingID.
			if searchID != "" {
				if searchID != tdoc.ID && searchID != rec.Instance {
					// keep looking
					return false
				}
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
			return stop || first
		})

	return dirRecs, dirTDs, deviceRecs, deviceTDs
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

// locateTDToUse attempts to locate a directory and returns its TDD.
//
// If a TD URL is offered then try to load the TD from that URL regardless
// if it is a device or directory. If this fails then return an error.
//
// If no URL is offered then search for a directory and return its TD.
//
// If no TDD can be found then return nil
func (cl *DiscoveryClientImpl) locateDirectory(tdURL string, maxWaitTime time.Duration) *td.TD {

	// if a URL is offered then work with it.
	if tdURL != "" {
		tdoc, _, err := LoadTD(tdURL, cl.rootCAs)
		if err != nil {
			slog.Warn("discoverDirectory: TD is not available at the provided URL",
				"tdURL", tdURL,
				"err", err.Error())
		}
		return tdoc
	}
	// no TD URL offered, so try to find a directory.
	tdoc := cl.DiscoverFirstTD("", discovery.DISCO_TYPE_DIRECTORY, maxWaitTime)

	return tdoc
}

// Locate the gateway TD to use.
// This uses thing discovery to search for a record of type "gateway".
// WoT doesn't specify the use of gateways so this is HiveOT only.
func (cl *DiscoveryClientImpl) locateGateway(maxWaitTime time.Duration) (gwTD *td.TD) {

	// if a TD URL is offered then work with it.
	// no URL offered or it doesn't work so go find one that can be read.
	gwTD = cl.DiscoverFirstTD("", discovery.DISCO_TYPE_GATEWAY, maxWaitTime)
	return gwTD
}

// NewDiscoveryClientImpl returns a ready-to-use discovery client.
//
// Call DiscoverThings or DiscoverDirectories to start the discovery process.
//
// If env has a directory TD URL then attempt to load this TD into the environment.
//
// If discoOnStart is enabled and no directory TD is known then attempt to discover
// a directory and update the application environment with the first directory found.
func NewDiscoveryClientImpl(
	env *api.HiveEnvironment, discoOnStart bool) (*DiscoveryClientImpl, error) {
	var err error

	thingID := discovery.DiscoveryClientCellType + "-" + shortid.MustGenerate()
	cl := &DiscoveryClientImpl{
		HiveCellBase:    cells.NewHiveCellBase(thingID),
		env:             env,
		discoverOnStart: discoOnStart,
	}
	// obtaining a server TD requires env for reading URLs and storing the TD
	if env == nil {
		return cl, nil
	}

	cl.rootCAs = env.GetRootCAs()

	// 1. If a directory TD URL is provided, download the TD
	dirTD := env.GetDirTD()
	if dirTD == nil && env.DirTDURL != "" {
		dirTD, _, err = LoadTD(env.DirTDURL, cl.rootCAs)
		if err == nil {
			slog.Info("NewDiscoveryClientImpl. Directory TD downloaded successfully",
				"dirURL", env.DirTDURL, "thingID", dirTD.ID)
			env.SetDirTD(dirTD)
		}
	}
	// 2. If no directory is provided but discoOnStart is set, then go look for one.
	if dirTD == nil && discoOnStart {
		dirTD = cl.DiscoverFirstTD("", discovery.DISCO_TYPE_DIRECTORY, time.Second)
		if dirTD != nil {
			slog.Info("NewDiscoveryClientImpl. Directory TD discovered successfully",
				"thingID", dirTD.ID)
			env.SetDirTD(dirTD)
		}
	}

	// 3. If a server TD URL is provided download it
	serverTD := env.GetServerTD()
	if serverTD == nil && env.ServerTDURL != "" {
		serverTD, _, err := LoadTD(env.ServerTDURL, cl.rootCAs)
		if err == nil {
			env.SetServerTD(serverTD)
			slog.Info("NewDiscoveryClientImpl. Server TD downloaded successfully",
				"thingID", serverTD.ID)
		}
	}

	var _ discovery.IDiscoveryClient = cl // interface check
	return cl, err
}

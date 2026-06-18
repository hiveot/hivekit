package discoverypkg

import (
	"crypto/x509"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/grandcat/zeroconf"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/factory"
	"github.com/hiveot/hivekit/go/modules/transport/discovery"
	internalclient "github.com/hiveot/hivekit/go/modules/transport/discovery/internal/client"
	tlsclientpkg "github.com/hiveot/hivekit/go/modules/transport/tlsclient/pkg"
)

// Client for discovery of WoT devices and directories
// When included in a module chain this auto-discovers a directory TDD and
// a gateway TD on Start.
// If an app environment is provided then it will set the serverURL if needed.
type DiscoveryClient struct {
	*modules.HiveModuleBase
	// optional update the discovery results in the app environment
	env *factory.AppEnvironment

	// discovery directory info when running DiscoveryFirstDirectory
	dirURL string // the directory TD instance
	// the discovered directory TD if available
	dirTD *td.TD

	// discovery of the server using the env serverURL
	serverURL string
	// the discovered server TD if available
	serverTD *td.TD

	// mux for access to discovered data
	mux sync.RWMutex
}

// DiscoverDirectories supports introduction mechanisms to bootstrap the WoT discovery
// process and returns a list of discovered directory TD URLs with the wot service name.
//
// Intended for clients that need to find one or more WoT directories.
//
//	instanceName is optional and intended to search for a particular instance by name, such as 'hub'.
//	duration is the time to search for.
//	firstResult stop scanning when the first result is received
//	cb is the callback to invoke when a match is found.
//
// This returns a list of all discoveries
func (cl *DiscoveryClient) DiscoverDirectories(
	instanceName string,
	maxWaitTime time.Duration,
	firstResult bool,
	cb func(*discovery.DiscoveryResult)) ([]*discovery.DiscoveryResult, error) {

	drList := make([]*discovery.DiscoveryResult, 0)

	// run the scan to collect results
	_, err := internalclient.DnsSDScan(instanceName, discovery.WOT_DIRECTORY_SERVICE_TYPE, maxWaitTime,
		func(rec *zeroconf.ServiceEntry) bool {

			// create a discovery record for the service entry
			discoRecord := cl.ParseZeroconfServiceEntry(rec)
			drList = append(drList, discoRecord)
			if cb != nil {
				cb(discoRecord)
			}
			return firstResult
		})
	return drList, err
}

// DiscoverDirectory downloads the first discovered TD directory
func (cl *DiscoveryClient) DiscoverFirstDirectory(
	instanceName string, searchTime time.Duration) (rec0 *discovery.DiscoveryResult, err error) {

	records, err := cl.DiscoverDirectories(instanceName, searchTime, true, nil)

	if len(records) == 0 {
		return nil, err
	}

	// Determine the directory URL and download the TD.
	rec0 = records[0]
	return rec0, nil
}

// DiscoverFirstGateway returns the discovery record if the first gateway server.
func (cl *DiscoveryClient) DiscoverFirstGateway(
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
func (cl *DiscoveryClient) DiscoverThings(
	instanceName string,
	maxWaitTime time.Duration,
	cb func(*discovery.DiscoveryResult) bool) ([]*discovery.DiscoveryResult, error) {

	var mux sync.RWMutex

	drList := make([]*discovery.DiscoveryResult, 0)

	// run the scan to collect results
	_, err := internalclient.DnsSDScan(instanceName, discovery.WOT_THING_SERVICE_TYPE, maxWaitTime,
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

// DownloadTD a TD document from the given URL.
// Intended for discovery of a directory TD.
//
// tdURL points to the discovery spec http well-known endpoint address. Only https is currently supported.
// caCert is optional CA to verify the server validity. nil skips this validation.
//
// This returns the TD JSON or an error if none is found
func (cl *DiscoveryClient) DownloadTD(tdURL string, caCert *x509.Certificate) (tdoc *td.TD, tdJSON string, err error) {

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
		return nil, "", err
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

// // Return the directory connection TD instance discovered during Start
func (cl *DiscoveryClient) GetDirectory() (dirTDD *td.TD) {
	cl.mux.RLock()
	defer cl.mux.RUnlock()

	return cl.dirTD
}

// Return the server TD instance discovered during Start
// // This returns nil if no server was discovered.
func (cl *DiscoveryClient) GetServerTD() *td.TD {
	cl.mux.RLock()
	defer cl.mux.RUnlock()

	return cl.serverTD
}

// Convert a zeroconf result to a hiveot discovery record
func (cl *DiscoveryClient) ParseZeroconfServiceEntry(
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
func (cl *DiscoveryClient) Start() (err error) {
	var dirURL string
	var serverURL string

	if cl.env != nil && cl.env.DirectoryURL != "" {
		dirURL = cl.env.DirectoryURL
	} else {
		rec0, err := cl.DiscoverFirstDirectory("", 0)
		if err == nil {
			dirURL = rec0.AsURL()
		}
	}
	// should this obtain the directory TD?
	if dirURL != "" {
		cl.dirTD, _, err = cl.DownloadTD(dirURL, cl.env.CaCert)
	}
	if err != nil {
		slog.Warn("Start: Unable to determine the directory URL. Directory is not available ..",
			"directoryURL", dirURL, "err", err.Error())
	}

	// FIXME: determine the server URL
	// what is a serverURL anyways? does this work without forms?
	//   RC doesn't need forms, so no need for a TD?...
	// what about connecting to a device?
	//   do devices expose their TD?
	//
	// option1: serverURL is hiveot transport endpoint. Just connect and send requests
	// option2: don't use serverURL. this is discovery so discover the URL
	if cl.env != nil && cl.env.ServerURL != "" {
		serverURL = cl.env.ServerURL
	} else {
		rec0, err := cl.DiscoverFirstGateway("", 0)
		if err != nil {
			serverURL = rec0.AsURL()
		}
	}
	if serverURL != "" {
		cl.serverTD, _, err = cl.DownloadTD(serverURL, cl.env.CaCert)
	}
	// optionally update the factory environment to share the results with other modules
	if cl.env != nil {
		if cl.env.DirectoryURL == "" {
			cl.env.DirectoryURL = dirURL
		}
		if cl.env.ServerURL == "" {
			cl.env.ServerURL = serverURL
		}
	}
	return nil
}

// NewDiscoveryClient creates a new instance of a discovery client
//
// appEnv is optional. On Start it will be updated with the discovered directory and server.
func NewDiscoveryClient(appEnv *factory.AppEnvironment) *DiscoveryClient {
	cl := &DiscoveryClient{
		HiveModuleBase: modules.NewHiveModuleBase("", 0),
		env:            appEnv,
	}
	return cl
}

// NewDiscoveryClientFactory creates a new instance of a discovery client for
// use by the factory.
// On start this updates the factory environment with the directory server URL.
//
// Intended to be used by a client side factory recipe to automatically discover the
// directory TDD and gateway TD.
func NewDiscoveryClientFactory(f factory.IModuleFactory) (modules.IHiveModule, error) {
	appEnv := f.GetEnvironment()
	cl := NewDiscoveryClient(appEnv)
	// nothing else to do here right now

	return cl, nil
}

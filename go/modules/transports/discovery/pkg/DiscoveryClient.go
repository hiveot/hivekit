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
	"github.com/hiveot/hivekit/go/modules/transports/discovery"
	"github.com/hiveot/hivekit/go/modules/transports/httpclient"
)

type DiscoveryResult struct {
	Addr        string // IP or hostname of the server
	Port        int    // port the server listens on
	IsDirectory bool   // URL is that of a Thing Directory
	IsThing     bool   // URL is of a Thing
	Instance    string
	// predefined WoT discovery parameters
	Schema string // Schema part of the URL
	Type   string // Thing or Directory
	TD     string // absolute pathname of the TD or TDD
	// hiveot connection endpoints
	AuthEndpoint string            // authentication service endpoint
	SSEEndpoint  string            // Http/SSE-SC transport protocol
	WSSEndpoint  string            // Websocket transport
	Params       map[string]string // optional parameters
}

// Return the URL contained in the discovery record
func (dr *DiscoveryResult) AsURL() string {
	fullUrl := fmt.Sprintf("%s://%s:%d%s", dr.Schema, dr.Addr, dr.Port, dr.TD)
	return fullUrl
}

// Client for discovery of WoT devices and directories
type DiscoveryClient struct {
}

// DiscoverFirstDirectory returns the first discovered TD directory
//
//	instanceName is the optional name of the service instance, "" for any
func (cl *DiscoveryClient) DiscoverFirstDirectory(instanceName string) (*DiscoveryResult, error) {

	records, err := cl.DiscoverDirectories(
		instanceName, time.Second*3, true, nil)

	if len(records) == 0 {
		return nil, err
	}
	return records[0], err
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
	cb func(*DiscoveryResult)) ([]*DiscoveryResult, error) {

	drList := make([]*DiscoveryResult, 0)

	// run the scan to collect results
	_, err := DnsSDScan(instanceName, discovery.WOT_DIRECTORY_SERVICE_TYPE, maxWaitTime,
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

// DiscoverThings returns discovery records of wot Things that publish themselves on the network.
//
// Intended for environments where things run servers themselves (instead of using a hub/gateway).
//
//	instanceName is optional and intended to search for a particular instance by name, such as 'hub'.
//	duration is the time to search for.
//	cb is the callback to invoke when a match is found.
//
// This returns a list of all discoveries
func (cl *DiscoveryClient) DiscoverThings(
	instanceName string,
	maxWaitTime time.Duration,
	cb func(*DiscoveryResult) bool) ([]*DiscoveryResult, error) {

	var mux sync.RWMutex

	drList := make([]*DiscoveryResult, 0)

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

// DownloadTDD the directory TD.
//
// if tddURL is empty then perform a discovery first.
// caCert is optional CA to verify the server validity. nil skips this validation.
//
// This returns the directory TD JSON or an error if none is found
func (cl *DiscoveryClient) DownloadTDD(tddURL string, caCert *x509.Certificate) (tddJSON string, err error) {

	// perform discovery if no URL is provided
	if tddURL == "" {
		record, err := cl.DiscoverFirstDirectory("")
		if err != nil {
			return "", fmt.Errorf("failed starting discovery: %w", err)
		}
		if record == nil {
			return "", fmt.Errorf("no TDD was discovered")
		}
		tddURL = fmt.Sprintf("%s://%s:%d%s", record.Schema, record.Addr, record.Port, record.TD)
	}

	parts, err := url.Parse(tddURL)
	if err != nil {
		return "", err
	}
	httpCl := httpclient.NewHttpClient(parts.Host, nil, caCert, 0)
	resp, statusCode, err := httpCl.Get(parts.Path)
	_ = statusCode
	if err != nil {
		return "", err
	}
	tddJSON = string(resp)
	return tddJSON, err
}

// Convert a zeroconf result to a hiveot discovery record
func (cl *DiscoveryClient) ParseZeroconfServiceEntry(rec *zeroconf.ServiceEntry) *DiscoveryResult {
	discoResult := DiscoveryResult{
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
	if rec.ServiceName() == discovery.WOT_DIRECTORY_SERVICE_TYPE {
		discoResult.IsDirectory = true
	} else if rec.ServiceName() == discovery.WOT_THING_SERVICE_TYPE {
		//this is a thing
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

// NewDiscoveryClient creates a new instance of a discovery client
func NewDiscoveryClient() *DiscoveryClient {
	cl := &DiscoveryClient{}
	return cl
}

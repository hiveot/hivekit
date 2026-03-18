// Package internal to publish Hub services for discovery
package internal

import (
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/grandcat/zeroconf"
	discoveryapi "github.com/hiveot/hivekit/go/modules/transports/discovery/api"
	"github.com/hiveot/hivekit/go/utils"
)

// ServeWotDiscovery publishes a TD Thing or Directory discovery record.
//
// WoT defines the directory as service type: _directory._sub._wot._tcp with the
// TXT record containing the fields 'td', 'type', and 'scheme'
// See also: https://w3c.github.io/wot-discovery/#introduction-dns-sd-sec
//
// endpoints is a hiveot addon that provides connection addresses for transport
// protocols without the need to use forms. This is intended for connection
// oriented protocols such as websocket, sse-sc, mqtt, and others. The schema
// identifies the protocol.
//
//	instanceName is the name of the server instance serving, like the module ID or "hiveot" for its hub.
//	  This can be used to search for a particular directory instance when multiple are available.
//	  When omitted, the device hostname is used.
//	tddURL is the URL the thing or directory TD is served at.
//	serviceType is WOT_THING_SERVICE_TYPE or WOT_DIRECTORY_SERVICE_TYPE
//	endpoints contains a map of additional {scheme:connection} connection URLs
//
// Returns the discovery service instance. Use Shutdown() when done.
func ServeWotDiscovery(
	instanceName string, tddURL string, serviceType string, endpoints map[string]string,
) (*zeroconf.Server, error) {

	parts, err := url.Parse(tddURL)
	if err != nil {
		return nil, err
	}

	// setup the introduction mechanism
	if instanceName == "" {
		instanceName, _ = os.Hostname()
	}
	tdPath := parts.Path
	if tdPath == "" {
		tdPath = discoveryapi.DefaultHttpGetDirectoryTDPath
	}
	portString := parts.Port()
	portNr, err := strconv.Atoi(portString)
	if err != nil {
		return nil, err
	}
	scheme := parts.Scheme
	address := parts.Hostname()
	if address == "127.0.0.1" || address == "localhost" {
		// DNS does not work for the local network. Use an external IP instead.
		outIP := utils.GetOutboundIP("")
		address = outIP.String()
		slog.Warn("ServeDirectoryDiscovery: TDD URL contains localhost address. "+
			"This doesn't work with DNS-SD discovery. Using external address instead",
			"addr", address)
	}
	// add WoT discovery parameters
	params := map[string]string{
		"td":     tdPath,
		"scheme": scheme,
		"type":   "Directory",
	}
	// add connection endpoints as parameters
	for ep, epURL := range endpoints {
		params[ep] = epURL
	}
	slog.Info("Serving discovery for address",
		slog.String("address", parts.Hostname()),
		slog.String("serviceType", serviceType),
		slog.Int("port", portNr),
	)
	discoServer, err := ServeDnsSD(
		instanceName, serviceType,
		address, portNr, params)

	return discoServer, err
}

// ServeDnsSD publishes a service discovery record.
//
//	DNS-SD will publish this as _{instance}._{serviceName}._tcp
//
//	instanceID is the unique ID of the service instance, usually the plugin-ID
//	serviceType is the discovery service type. Eg: _wot._tcp.
//	address service listening IP address
//	port service listing port
//	params is a map of key-value pairs to include in discovery, eg td, type and scheme in wot
//
// Returns the discovery service instance. Use Shutdown() when done.
func ServeDnsSD(instanceID string, serviceType string,
	address string, port int, params map[string]string) (*zeroconf.Server, error) {
	var ips []string

	slog.Info("ServeDnsSD",
		slog.String("instanceID", instanceID),
		slog.String("serviceType", serviceType),
		slog.String("address", address),
		slog.Int("port", port),
		"params", params)
	if serviceType == "" {
		err := fmt.Errorf("Empty serviceType")
		return nil, err
	}

	// only the local domain is supported
	domain := "local."
	hostname, _ := os.Hostname()

	// if the given address isn't a valid IP address. try to resolve it instead
	ips = []string{address}
	if net.ParseIP(address) == nil {
		// was a hostname provided instead IP?
		hostname = address
		parts := strings.Split(address, ":") // remove port
		actualIP, err := net.LookupIP(parts[0])
		if err != nil {
			// can't continue without a valid address
			slog.Error("ServeDnsSD: Provided address is not an IP and cannot be resolved",
				"address", address, "err", err)
			return nil, err
		}
		ips = []string{actualIP[0].String()}
	}

	ifaces, err := utils.GetInterfaces(ips[0])
	if err != nil || len(ifaces) == 0 {
		slog.Warn("ServeDnsSD: Address does not appear on any interface. Continuing anyways", "address", ips[0])
	}
	// add a text record with key=value pairs
	textRecord := []string{}
	for k, v := range params {
		textRecord = append(textRecord, fmt.Sprintf("%s=%s", k, v))
	}
	server, err := zeroconf.RegisterProxy(
		instanceID, serviceType, domain, int(port), hostname, ips, textRecord, ifaces)
	if err != nil {
		slog.Error("ServeDnsSD: Failed to start the zeroconf server", "err", err)
	}
	return server, err
}

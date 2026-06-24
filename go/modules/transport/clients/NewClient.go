// Package clients containing all clients.
// Only use this if you wish to include all protocol clients, which adds around 10MB
package clients

import (
	"crypto/x509"
	"fmt"
	"net/url"
	"strings"

	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/factory"
	"github.com/hiveot/hivekit/go/modules/transport"
	grpcpkg "github.com/hiveot/hivekit/go/modules/transport/grpc/pkg"
	httpbasicpkg "github.com/hiveot/hivekit/go/modules/transport/httpbasic/pkg"
	ssescpkg "github.com/hiveot/hivekit/go/modules/transport/ssesc/pkg"
	wsspkg "github.com/hiveot/hivekit/go/modules/transport/wss/pkg"
)

// Module type for inclusion in the factory chain
const TransportClientModuleType = "transport-client"

// list of supported client protocols
var SupportedClientProtocols = []string{
	transport.ProtocolSchemeHiveotGrpc,
	transport.ProtocolSchemeHiveotSseSc,
	transport.SubprotocolHiveotWebsocket,
	transport.SubprotocolWotWebsocket,
	transport.ProtocolSchemeWotHttpBasic,
	// transport.ProtocolSchemeWotMqtt,
}

// GetProtocolType returns the protocol used for connecting to this device.
// This returns the protocol type and connection href, if available.
//
// Not intended to get the href of an operation as a subprotocol can use a different
// connection protocol for the return channel, eg, SSE.
//
// Note that TDs can use multiple protocols for its operations. HiveOT currently assumes
// that only a single protocol is used for connecting with a device. Steps:
//
//  1. If a base is present then use that as the href
//
//  2. if an operation is provided then lookup the form for that operation
//     if no base is provided use the href of the operation
//
//  3. if the operation has a subprotocol then use this as the protocol-type
//
//  4. if no subprotocol is provided in the operation then derive it from href
func GetProtocolType(tdoc *td.TD, op string) (protocolType string, href string) {
	subprotocol := ""
	// 1. derive href  from base
	if tdoc.Base != "" {
		href = tdoc.Base
	}
	if op != "" {
		// 2. if an op is provided determine href and subprotocol from the form
		forms := tdoc.GetForms(op, "")
		if len(forms) > 0 {
			form := forms[0]
			if href == "" {
				href = form.GetHRef()
			}
			subprotocol, _ = form.GetSubprotocol()
		}
	}
	// 3. determine the protocol type from the subprotocol
	switch subprotocol {
	case transport.SubprotocolHiveotSsesc:
		protocolType = transport.ProtocolTypeHiveotSsesc
	case transport.SubprotocolHiveotWebsocket:
		protocolType = transport.ProtocolTypeHiveotWebsocket
	case transport.SubprotocolWotWebsocket:
		protocolType = transport.ProtocolTypeWotWebsocket
	case transport.SubprotocolWotHttpLongPoll:
		protocolType = transport.ProtocolTypeWotHttpLongPoll
	}

	// if a subprotocol is found then use it
	if protocolType != "" {
		return protocolType, href
	}

	// 4. no subprotocol is provided, derive it from the URI Scheme
	parts, err := url.Parse(href)
	if err != nil {
		return "", ""
	}
	scheme := strings.ToLower(parts.Scheme)
	switch scheme {
	case transport.ProtocolSchemeHiveotGrpc:
		protocolType = transport.ProtocolTypeHiveotGrpc
	case transport.ProtocolSchemeWotHttpBasic:
		protocolType = transport.ProtocolTypeWotHttpBasic
	case transport.ProtocolSchemeWotWebsocket:
		protocolType = transport.ProtocolTypeWotWebsocket
	case transport.ProtocolSchemeWotMqtt:
		protocolType = transport.ProtocolTypeWotMqtt
	case transport.ProtocolSchemeWotSse:
		protocolType = transport.ProtocolTypeWotSse
	default:
		protocolType = scheme
	}
	return protocolType, href
}

// NewTransportClient returns a new client module instance ready to connect to a transport server
// using the given URL.
//
//	protocolType provides direct control of the client to create regardless of the URL.
//	 If omitted, then it is derived from the serverURL scheme.
//
//	serverURL is the connection endpoint to connect to
//
//	caCert is the CA certificate to validate the server certificate.
//
// # Use SetTimeout for increasing the default communication timeout for testing
//
// This is intended to be used as a sink for application modules.
func NewTransportClient(
	protocolType string, serverURL string, caCert *x509.Certificate) (
	cl transport.ITransportClient, err error) {

	// // 1. determine the connection address
	// if serverURL == "" {
	// 	// use the first hiveot instance to connect to
	// 	discoClient := discoverypkg.NewDiscoveryClient()
	// 	discoList, err := discoClient.DiscoverThings(discoserver.DefaultServiceName, timeout, nil)
	// 	if err != nil || len(discoList) == 0 {
	// 		return nil, fmt.Errorf("no server found")
	// 	}
	// 	// match protocolType
	// 	serverURL = discoList[0].WSSEndpoint
	// 	if serverURL == "" {
	// 		serverURL = discoList[0].SSEEndpoint
	// 	}
	// }

	parts, err := url.Parse(serverURL)
	scheme := strings.ToLower(parts.Scheme)
	// the protocol to use is based on scheme

	// use the URL to determine the protocol
	if protocolType == "" {
		scheme := strings.ToLower(parts.Scheme)
		switch scheme {
		case transport.ProtocolSchemeHiveotGrpc:
			protocolType = transport.ProtocolTypeHiveotGrpc
		case transport.ProtocolSchemeWotHttpBasic:
			protocolType = transport.ProtocolTypeWotHttpBasic
		case transport.ProtocolSchemeWotWebsocket:
			protocolType = transport.ProtocolTypeWotWebsocket
		case transport.ProtocolSchemeWotMqtt:
			protocolType = transport.ProtocolTypeWotMqtt
		case transport.ProtocolSchemeWotSse:
			protocolType = transport.ProtocolTypeWotSse
		default:
			protocolType = scheme
		}
	}

	switch protocolType {
	case transport.ProtocolTypeHiveotGrpc:
		// don't use TLS on unix domain sockets
		// if strings.HasPrefix(serverURL, "unix") {
		// 	caCert = nil
		// }
		cl = grpcpkg.NewHiveotGrpcClient(serverURL, caCert)

	case transport.ProtocolTypeHiveotSsesc:
		cl = ssescpkg.NewSseScClient(serverURL, caCert)

	case transport.ProtocolTypeHiveotWebsocket:
		cl = wsspkg.NewHiveotWssClient(serverURL, caCert)

	case transport.ProtocolTypeWotWebsocket:
		cl = wsspkg.NewWotWssClient(serverURL, caCert)

	case transport.ProtocolTypeWotHttpBasic:
		caCert := caCert
		cl = httpbasicpkg.NewHttpBasicClient(serverURL, caCert, nil)

	//case transport.ProtocolTypeWotMQTTWSS:
	//	fullURL = testServerMqttWssURL
	default:
		err = fmt.Errorf("NewTransportClient. Unknown protocol '%s'", scheme)
	}
	return cl, err
}

// NewTransportClientFromTD returns a new client module instance ready to connect to a
// thing.
//
// This uses the TD base to determine the connection protocol.
func NewTransportClientFromTD(
	tdoc *td.TD, caCert *x509.Certificate) (cl transport.ITransportClient, err error) {

	protocolType, href := GetProtocolType(tdoc, "")
	cl, err = NewTransportClient(protocolType, href, caCert)
	return cl, err
}

// Create a new client instance using the gathered information from the factory
// This uses the factory serverURL or server TD to determine which protocol to instantiate
func NewTransportClientFactory(f factory.IModuleFactory,
	md *factory.ModuleDefinition) (cl modules.IHiveModule, err error) {

	serverURL := f.GetEnvironment().ServerURL
	if serverURL != "" {
		cl, err = NewTransportClient("", serverURL, f.GetEnvironment().CaCert)
	} else {
		// TODO: use discovered server TD
		err = fmt.Errorf("unknown protocol")
	}
	return cl, err
}

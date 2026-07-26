// Package clients containing all clients.
// Only use this if you wish to include all protocol clients, which adds around 10MB
package clients

import (
	"crypto/x509"
	"fmt"
	"net/url"
	"strings"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/td"
	grpc_client "github.com/hiveot/hivekit/go/modules/transport/grpc/client"
	httpbasic_client "github.com/hiveot/hivekit/go/modules/transport/httpbasic/client"
	ssesc_client "github.com/hiveot/hivekit/go/modules/transport/ssesc/client"
	wss_client "github.com/hiveot/hivekit/go/modules/transport/wss/client"
)

// Module type for inclusion in the factory chain
const TransportClientModuleType = "transport-client"

// list of supported client protocols
var SupportedClientProtocols = []string{
	api.ProtocolSchemeHiveotGrpc,
	api.ProtocolSchemeHiveotSseSc,
	api.SubprotocolHiveotWebsocket,
	api.SubprotocolWotWebsocket,
	api.ProtocolSchemeWotHttpBasic,
	// api.ProtocolSchemeWotMqtt,
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
func GetProtocolType(tdoc *td.TD, op string, name string) (protocolType string, href string) {
	subprotocol := ""
	// 1. derive href  from base
	if tdoc.Base != "" {
		href = tdoc.Base
	}
	// if an operation is given then its form will provide an updated href and potentially a subprotocol
	if op != "" {
		// 2. if an op is provided determine href and subprotocol from the form
		forms := tdoc.GetForms(op, name)
		if len(forms) > 0 {
			form := forms[0]
			if href == "" {
				href = form.GetHRef()
			}
			subprotocol, _ = form.GetSubprotocol()
		}
	} else {
		// no operation, so just need a connection. the scheme is
	}

	// 3. determine the protocol type from the subprotocol
	switch subprotocol {
	case api.SubprotocolHiveotSsesc:
		protocolType = api.ProtocolTypeHiveotSsesc
	case api.SubprotocolHiveotWebsocket:
		protocolType = api.ProtocolTypeHiveotWebsocket
	case api.SubprotocolWotWebsocket:
		protocolType = api.ProtocolTypeWotWebsocket
	case api.SubprotocolWotHttpLongPoll:
		protocolType = api.ProtocolTypeWotHttpLongPoll
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
	case api.ProtocolSchemeHiveotGrpc:
		protocolType = api.ProtocolTypeHiveotGrpc
	case api.ProtocolSchemeWotHttpBasic:
		protocolType = api.ProtocolTypeWotHttpBasic
	case api.ProtocolSchemeWotWebsocket:
		protocolType = api.ProtocolTypeWotWebsocket
	case api.ProtocolSchemeWotMqtt:
		protocolType = api.ProtocolTypeWotMqtt
	case api.ProtocolSchemeWotSse:
		protocolType = api.ProtocolTypeWotSse
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
//	serverURL is the connection endpoint to connect to.
//		note that without a TD, http-basic cannot be used.
//
//	caCert is the CA certificate to validate the server certificate.
//
// # Use SetTimeout for increasing the default communication timeout for testing
//
// This is intended to be used as a sink for application modules.
func NewTransportClient(
	protocolType string, serverURL string, caCert *x509.Certificate) (
	cl api.ITransportClient, err error) {

	parts, err := url.Parse(serverURL)
	scheme := strings.ToLower(parts.Scheme)
	// the protocol to use is based on scheme

	// use the URL to determine the protocol
	if protocolType == "" {
		scheme := strings.ToLower(parts.Scheme)
		switch scheme {
		case api.ProtocolSchemeHiveotGrpc:
			protocolType = api.ProtocolTypeHiveotGrpc
		// hiveot sse doesn't need a TD as the protocol is self-descriptive
		case api.ProtocolSchemeHiveotSseSc:
			protocolType = api.ProtocolTypeHiveotSsesc

		case api.ProtocolSchemeWotHttpBasic:
			protocolType = api.ProtocolTypeWotHttpBasic
		case api.ProtocolSchemeWotWebsocket:
			protocolType = api.ProtocolTypeWotWebsocket
		case api.ProtocolSchemeWotMqtt:
			protocolType = api.ProtocolTypeWotMqtt
		// case api.ProtocolSchemeWotSse:
		// 	protocolType = api.ProtocolTypeWotSse
		default:
			protocolType = scheme
		}
	}

	switch protocolType {
	case api.ProtocolTypeHiveotGrpc:
		// don't use TLS on unix domain sockets
		// if strings.HasPrefix(serverURL, "unix") {
		// 	caCert = nil
		// }
		cl = grpc_client.NewHiveotGrpcClient(serverURL, caCert)

	case api.ProtocolTypeHiveotSsesc:
		cl = ssesc_client.NewSseScClient(serverURL, caCert)

	case api.ProtocolTypeHiveotWebsocket:
		cl = wss_client.NewHiveotWssClient(serverURL, caCert)

	case api.ProtocolTypeWotWebsocket:
		cl = wss_client.NewWotWssClient(serverURL, caCert)

	case api.ProtocolTypeWotHttpBasic:
		// without a TD http-basic cannot be used
		err = fmt.Errorf("NewTransportClient. Cannot use URL '%s'. Http-basic requires a TD ", serverURL)
		// cl = httpbasic_client.NewHttpBasicClient(serverURL, caCert, tdoc)

	//case api.ProtocolTypeWotMQTTWSS:
	//	fullURL = testServerMqttWssURL
	default:
		err = fmt.Errorf("NewTransportClient. Unsupported URL scheme '%s'", scheme)
	}
	return cl, err
}

// NewTransportClientFromForm returns a new client module instance needed to invoke
// the operation in the form.
//
// This uses the TD, op and name to determine the connection protocol.
//
// If name is empty then op should be a thing level operation
// If op is empty then this uses the TD 'base' as the connection URL
func NewTransportClientFromTD(
	tdoc *td.TD, op string, name string, caCert *x509.Certificate) (cl api.ITransportClient, err error) {

	protocolType, href := GetProtocolType(tdoc, op, name)

	switch protocolType {
	case api.ProtocolTypeHiveotGrpc:
		// don't use TLS on unix domain sockets
		// if strings.HasPrefix(serverURL, "unix") {
		// 	caCert = nil
		// }
		cl = grpc_client.NewHiveotGrpcClient(href, caCert)

	case api.ProtocolTypeHiveotSsesc:
		cl = ssesc_client.NewSseScClient(href, caCert)

	case api.ProtocolTypeHiveotWebsocket:
		cl = wss_client.NewHiveotWssClient(href, caCert)

	case api.ProtocolTypeWotWebsocket:
		cl = wss_client.NewWotWssClient(href, caCert)

	case api.ProtocolTypeWotHttpBasic:
		caCert := caCert
		cl = httpbasic_client.NewHttpBasicClient(tdoc, caCert)

	//case api.ProtocolTypeWotMQTTWSS:
	//	fullURL = testServerMqttWssURL
	default:
		err = fmt.Errorf("NewTransportClient. Unknown or missing protocol type '%s' for href '%s'",
			protocolType, href)
	}
	return cl, err
}

// Create a new client instance using the factory app environment.
//
// This requires the app environment server TD. This can be provided manually or through client discovery.
// If no Server TD is set in the app environment, try the server URL as fallback.
//
// Intended for RC and consumer recipes that connect to a gateway, or a consumer of a single device
// without using a directory.
func NewTransportClientFactory(
	f api.IModuleFactory, md *api.ModuleDefinition) (cl api.IHiveModule, err error) {

	env := f.GetEnvironment()

	// the server url is set through commandline, or useing a discovery client
	if env.ServerTD != nil {
		// there is no op or name to use so this requires the TD to have a 'base' URL
		cl, err = NewTransportClientFromTD(env.ServerTD, "", "", env.CaCert)
	} else if env.ServerURL != "" {
		cl, err = NewTransportClient("", env.ServerURL, env.CaCert)
	} else {
		// TODO: use discovered server TD
		err = fmt.Errorf("NewTransportClientFactory: no server TD or URL is available")
	}
	return cl, err
}

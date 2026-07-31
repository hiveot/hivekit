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
	api.HiveotGrpcUnixScheme,
	api.HiveotSseScScheme,
	api.HiveotWebsocketSubprotocol,
	api.WotWebsocketSubprotocol,
	api.HttpBasicScheme,
	// api.ProtocolSchemeWotMqtt,
}

// Derive the protocol type the form href and subprotocol.
func GetFormProtocolType(tdoc *td.TD, form *td.Form) (protocolType string, href string, err error) {

	hrefURL, err := form.ResolveHRef(tdoc.Base, nil)
	if err != nil {
		return "", href, err
	}

	scheme := strings.ToLower(hrefURL.Scheme)
	subprotocol, _ := form.GetSubprotocol()
	protocolType = scheme + ":" + subprotocol
	href = hrefURL.String()

	return protocolType, href, nil
}

// GetProtocolType returns the protocol used for connecting to the TD server.
// This returns the protocol type, connection href and form, if available.
//
// Note that TDs can include multiple protocols for its operations. For this
// purpose, an optional 'preferred' protocol can help narrow down the selection.
// This defaults to websocket.
//
// Steps:
//
//  1. Lookup the available forms that match the operation.
//     If no operation is provided then all thing level forms are returned.
//
//  2. Determine the protocol type of each form. If form with a preferred type is found, use it.
//
//  3. Get the form href. If href is relative then use the TD base to construct an absolute href.
//
//     tdoc whose forms to look up
//     op optional operation to locate
//     name optional affordance name to locate
//     preferred the preferred protocol type to use, or "" for any. This defaults to websocket.
//
// This returns the protocol type name, the href and the form that defined it.
// The form can contain additional information depending on the protocol.
func GetProtocolType(tdoc *td.TD, op string, name string, preferred string) (
	protocolType string, href string, form *td.Form) {

	var forms []td.Form
	// var subprotocol = ""

	if preferred == "" {
		preferred = api.WotWebsocketProtocolType
	}

	// 1. derive href  from base
	if tdoc.Base != "" {
		href = tdoc.Base
	}
	// if an operation is given then its form will provide an updated href and potentially a subprotocol
	if op != "" {
		// 2. if an op is provided determine href and subprotocol from the form
		forms = tdoc.GetForms(op, name)
	} else {
		// no operation, so the caller just needs a connection. Use the thing level forms.
		forms = tdoc.Forms
	}
	// iterate available forms and determine the best protocol type
	for _, f := range forms {
		ptype, fhref, _ := GetFormProtocolType(tdoc, &f)
		if ptype == preferred {
			form = &f
			href = fhref
			protocolType = ptype
			break
		}
	}
	// if no preferred protocol was found, use the first one
	if form == nil && len(forms) > 0 {
		form = &forms[0]
		protocolType, href, _ = GetFormProtocolType(tdoc, form)
	}

	return protocolType, href, form
}

// NewTransportClient returns a new client module instance ready to connect to a
// transport server using the given TD, operation and optional affordance name.
func NewTransportClient(
	tdoc *td.TD, op string, name string, caCert *x509.Certificate) (
	cl api.ITransportClient, err error) {

	form, _ := tdoc.GetForm(op, name, "", "")
	cl, err = NewTransportClientFromForm(tdoc, form, caCert)
	return cl, err
}

// NewTransportClientFromForm returns a new client module instance ready to connect to a
// transport server using the given form.
func NewTransportClientFromForm(
	tdoc *td.TD, form *td.Form, caCert *x509.Certificate) (
	cl api.ITransportClient, err error) {

	hrefURL, err := form.ResolveHRef(tdoc.Base, nil)
	href := hrefURL.String()
	subProtocol, _ := form.GetSubprotocol()

	// Note: the definition of protocol-type is scheme:subprotocol
	protocolType := hrefURL.Scheme + ":" + subProtocol

	switch protocolType {
	case api.HiveotGrpcTcpProtocolType:
		// gRPC only needs a connect href. All operations use this connection.
		cl = grpc_client.NewHiveotGrpcClient(href, caCert)

	case api.HiveotGrpcUnixProtocolType:
		// gRPC only needs a connect href. All operations use this connection.
		cl = grpc_client.NewHiveotGrpcClient(href, caCert)

	case api.HiveotSseScProtocolType:
		// SSE-SC has one full href to connect with SSE and 3 well-known relative paths to
		// receive requests, response and notification messages. The SSE channel
		// passes these in reverse direction.
		cl = ssesc_client.NewSseScClient(tdoc.Base, caCert)

	case api.HiveotWebsocketProtocolType:
		// websockets only needs a connect href. All operations use this connection.
		cl = wss_client.NewHiveotWssClient(href, caCert)

	case api.WotWebsocketProtocolType:
		// websockets only needs a connect href. All operations use this connection.
		cl = wss_client.NewWotWssClient(href, caCert)

	case api.HttpBasicProtocolType:
		// http-basic needs the TD to get a href per operation.
		cl = httpbasic_client.NewHttpBasicClient(tdoc, caCert)

		//case api.ProtocolTypeWotMQTTWSS:
		// mqtt needs the TD for href to connect and mkv:topic field for topics per operation
		// cl = mqtt_client.NewMqttsClient(tdoc, caCert)

	default:
		err = fmt.Errorf("NewTransportClient. Unsupported protocol type '%s'", protocolType)
	}
	return cl, err
}

// NewFallbackTransportClient returns a new client module instance ready to
// connect to a transport server using the URL scheme.
//
// Note that this does not support http-basic or mqtt as these protocols need
// a TD to determine the href per operation.
// This does work for wot websockets, hiveot grpc connections and sse-sc connections
//
// Note-1: if a scheme has multiple subprotocols then the most generic protocol
// will be chosen. This is a last-resort way to create a client.
func NewFallbackTransportClient(serverURL string, caCert *x509.Certificate) (
	cl api.ITransportClient, err error) {

	parts, err := url.Parse(serverURL)

	switch parts.Scheme {
	case api.HiveotGrpcUnixScheme:
		cl = grpc_client.NewHiveotGrpcClient(serverURL, caCert)

	case api.HiveotGrpcTcpScheme:
		cl = grpc_client.NewHiveotGrpcClient(serverURL, caCert)

	case api.HiveotSseScScheme:
		cl = ssesc_client.NewSseScClient(serverURL, caCert)

	case api.WotWebsocketScheme:
		cl = wss_client.NewWotWssClient(serverURL, caCert)

	default:
		err = fmt.Errorf("NewTransportClient. Unsupported protocol for URL '%s'", serverURL)
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
	tdoc := env.ServerTD

	// the server url is set through commandline, or useing a discovery client
	if tdoc != nil {
		// prefer the wot websocket if available
		form, _ := tdoc.GetConnectForm(api.WotWebsocketScheme, api.WotWebsocketSubprotocol)

		// there is no op or name to use so this requires the TD to have a 'base' URL
		cl, err = NewTransportClientFromForm(tdoc, form, env.CaCert)
	} else if env.ServerURL != "" {
		cl, err = NewFallbackTransportClient(env.ServerURL, env.CaCert)
	} else {
		// TODO: use discovered server TD
		err = fmt.Errorf("NewTransportClientFactory: no server TD or URL is available")
	}
	return cl, err
}

// Package clients containing all clients.
// Only use this if you wish to include all protocol clients, which adds around 10MB
package clients

import (
	"crypto/x509"
	"fmt"
	"log/slog"
	"net/url"
	"strings"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/td"
	grpc_client "github.com/hiveot/hivekit/go/cells/transport/grpc/client"
	httpbasic_client "github.com/hiveot/hivekit/go/cells/transport/httpbasic/client"
	ssesc_client "github.com/hiveot/hivekit/go/cells/transport/ssesc/client"
	wss_client "github.com/hiveot/hivekit/go/cells/transport/wss/client"
)

// Cell type for inclusion in the cell factory
const TransportClientCellType = "transport-client"

// list of supported client protocols
var SupportedClientProtocols = []string{
	api.HiveotGrpcUnixScheme,
	api.HiveotSseScScheme,
	api.HiveotWebsocketSubprotocol,
	api.WotWebsocketSubprotocol,
	api.WotHttpBasicScheme,
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

// NewTransportClient returns a new transport client instance ready to connect to a
// transport server using the given TD, operation and optional affordance name.
func NewTransportClient(
	tdoc *td.TD, op string, name string, rootCAs *x509.CertPool) (
	cl api.ITransportClient, err error) {

	form, _ := tdoc.GetForm(op, name, "", "")
	cl, err = NewTransportClientFromForm(tdoc, form, rootCAs)
	return cl, err
}

// NewTransportClientFromForm returns a new transport client instance ready to connect to a
// transport server using the given form.
func NewTransportClientFromForm(
	tdoc *td.TD, form *td.Form, rootCAs *x509.CertPool) (
	cl api.ITransportClient, err error) {

	hrefURL, err := form.ResolveHRef(tdoc.Base, nil)
	href := hrefURL.String()
	subProtocol, _ := form.GetSubprotocol()

	// Http basic needs a TD to invoke operations
	if subProtocol == "" && hrefURL.Scheme == api.WotHttpBasicScheme {
		cl, err = httpbasic_client.NewHttpBasicClient(tdoc, rootCAs)
	} else if subProtocol == api.HiveotSsescSubprotocol && hrefURL.Scheme == api.HiveotSseScScheme {
		cl = ssesc_client.NewSseScClient(tdoc, rootCAs)
	} else {
		// connection based protocols just need the href and subprotocol
		cl, err = NewTransportClientFromURL(href, subProtocol, rootCAs)
	}

	return cl, err
}

// NewTransportClientFromURL returns a new connection based transport client
// instance ready to connect to a transport server using the URL scheme.
//
// This does not support http-basic, as http-basic needs a form per operation.
//
//	serverURL full URL to connect to.
//	subProtocol use for protocol selection from forms, or "" to use URL scheme.
func NewTransportClientFromURL(serverURL string, subProtocol string, rootCAs *x509.CertPool) (
	cl api.ITransportClient, err error) {

	var protocolType string
	// 1. determine the protocol to use
	parts, err := url.Parse(serverURL)
	if subProtocol != "" {
		protocolType = parts.Scheme + ":" + subProtocol
	} else {
		// map the scheme to the protocol type
		switch parts.Scheme {
		case api.HiveotGrpcUnixScheme:
			protocolType = api.HiveotGrpcUnixProtocolType
		case api.HiveotGrpcTcpScheme:
			protocolType = api.HiveotGrpcTcpProtocolType
		case api.HiveotSseScScheme:
			protocolType = api.HiveotSseScProtocolType
		case api.WotHttpBasicScheme:
			protocolType = api.WotHttpBasicProtocolType
		case api.WotWebsocketScheme:
			protocolType = api.WotWebsocketProtocolType
		default:
			err = fmt.Errorf("NewTransportClient. Unsupported protocol type '%s'", protocolType)
		}
	}

	switch protocolType {
	case api.HiveotGrpcTcpProtocolType:
		// gRPC only needs a connect href. All operations use this connection.
		cl = grpc_client.NewHiveotGrpcClient(serverURL, rootCAs)

	case api.HiveotGrpcUnixProtocolType:
		// gRPC only needs a connect href. All operations use this connection.
		cl = grpc_client.NewHiveotGrpcClient(serverURL, rootCAs)

	// case api.HiveotSseScProtocolType:
	// 	// SSE-SC defines the SSE connect URL in 'base' and specifies 3 relative paths
	// 	// to send requests, response and notification messages to the server over http.
	// 	// The server sends these messages over the SSE channel.
	// 	cl = ssesc_client.NewSseScClient(tdoc, rootCAs)

	case api.HiveotWebsocketProtocolType:
		// websockets only needs a connect href. All operations use this connection.
		cl = wss_client.NewHiveotWssClient(serverURL, rootCAs)

	case api.WotWebsocketProtocolType:
		// websockets only needs a connect href. All operations use this connection.
		cl = wss_client.NewWotWssClient(serverURL, rootCAs)

	// case api.WotHttpBasicProtocolType:
	// http-basic needs the TD to get a href per operation.
	// cl, err = httpbasic_client.NewHttpBasicClient(tdoc, rootCAs)

	default:
		err = fmt.Errorf("NewTransportClient. Unsupported protocol for URL '%s'", serverURL)
	}
	return cl, err
}

// Create a new ready-to-use gateway client instance using the factory environment.
//
// Intended for RC and consumer recipes for using a gateway connection.
//
// This cell is only useful together with a directory TD in the environment because
// it needs the connection protocol type and destination. The directory connection
// is that of the gateway.
// Note that this does not use the directory http discovery endpoint. Instead the
// connection uses a bidirectional protocol like websocket or equivalent.
//
// This cell can be followed in the chain with a router cell to support connecting to
// stand-alone devices.
//
// The HiveEnvironment client token or cert is applied to the client.
func NewTransportClientFactory(
	f api.ICellFactory, md *api.CellDefinition) (cl api.IHiveCell, err error) {

	var tpcl api.ITransportClient
	env := f.GetEnvironment()

	// if a server URL override is used then ignore the directory TD
	if env.ServerURL != "" {
		tpcl, err = NewTransportClientFromURL(env.ServerURL, "", env.GetRootCAs())
	} else {

		dirTD := env.GetDirTD()

		if dirTD != nil {
			// pick a form with websocket support, if available
			form, _ := dirTD.GetForm("", "", api.WotWebsocketScheme, api.WotWebsocketSubprotocol)
			tpcl, err = NewTransportClientFromForm(dirTD, form, env.GetRootCAs())
		} else {
			// Give up. Cannot establish a gateway connection without the directory TD.
			slog.Warn("NewTransportClientFactory: no directory TD found so this cell is not activated")
			return nil, nil
		}
	}
	// set auth token and cert, when available, and activate the connection.
	token, _ := env.GetAuthToken()
	if token != "" {
		err = tpcl.SetAuthToken(env.ClientID, token, td.SecSchemeBearer)
	}
	clientCert, _ := env.GetClientCert()
	if clientCert != nil {
		tpcl.SetClientCert(clientCert)
	}
	err = tpcl.Connect()
	return tpcl, err
}

package clients

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/url"
	"strings"

	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules/transports"
	grpctransportpkg "github.com/hiveot/hivekit/go/modules/transports/grpc/pkg"
	httpbasicpkg "github.com/hiveot/hivekit/go/modules/transports/httpbasic/pkg"
	ssescpkg "github.com/hiveot/hivekit/go/modules/transports/ssesc/pkg"
	wsspkg "github.com/hiveot/hivekit/go/modules/transports/wss1/pkg"
)

// GetProtocolType returns the protocol used for connecting to this device.
// This returns the protocol type and connection href, if available.
//
// Note that TDs can use multiple protocols for its operations. HiveOT currently assumes
// that only a single protocol is used for connecting with a device. Steps:
//
//  1. lookup the forms of the 'observeallproperties' or 'subscribeallevents' top level operation,
//     if present take the first form and lookup the subprotocol and href fields.
//     if no subprotocol field exists, then use the scheme in the href.
//
//  2. if no form was found then use the scheme of the 'base' field in the td.
//     wss maps to websocket, subprotocol can be longpoll, sse is hiveot sse-sc, etc.
//
// 3. if all else fails if still no scheme then assume its http basic
func GetProtocolType(tdoc *td.TD) (protocolType string, href string) {
	forms := tdoc.GetForms(td.OpObserveAllProperties, "")
	if len(forms) == 0 {
		forms = tdoc.GetForms(td.OpSubscribeAllEvents, "")
	}
	if len(forms) == 0 {
		forms = tdoc.Forms // pick any
	}
	if len(forms) > 0 {
		subprotocol, found := forms[0].GetSubprotocol()
		href = forms[0].GetHRef()
		parts, err := url.Parse(href)
		// if href has no scheme then join with the 'base' field
		if err == nil && parts.Scheme == "" && tdoc.Base != "" {
			href, err = url.JoinPath(tdoc.Base, href)
		}
		if found {
			switch subprotocol {
			case transports.SubprotocolHiveotSsesc:
				protocolType = transports.ProtocolTypeHiveotSsesc
			case transports.SubprotocolHiveotWebsocket:
				protocolType = transports.ProtocolTypeHiveotWebsocket
			case transports.SubprotocolWotWebsocket:
				protocolType = transports.ProtocolTypeWotWebsocket
			case transports.SubprotocolWotHttpLongPoll:
				protocolType = transports.ProtocolTypeWotHttpLongPoll
			}
		}
	}
	if protocolType != "" {
		return protocolType, href
	}
	// if no subprotocol was found determine it from the href
	if href == "" {
		href = tdoc.Base
	}
	if strings.HasPrefix(href, transports.UriSchemeWotHttpBasic) {
		return transports.ProtocolTypeWotHttpBasic, href
	}
	// a normal TD device should have a subprotocol so not sure what is going on here.
	// just some fallback options
	if strings.HasPrefix(href, transports.UriSchemeWotWebsocket) {
		return transports.ProtocolTypeWotWebsocket, href
	}
	if strings.HasPrefix(href, transports.UriSchemeWotMqtt) {
		return transports.ProtocolTypeWotMqtt, href
	}
	if strings.HasPrefix(href, transports.UriSchemeWotSse) {
		return transports.ProtocolTypeWotSse, href
	}
	if strings.HasPrefix(href, transports.UriSchemeHiveotGrpc) {
		return transports.ProtocolTypeHiveotGrpc, href
	}
	return "", href
}

// NewTransportClient returns a new client module instance ready to connect to a transport server
// using the given URL.
//
//	protocolType provides direct control of the client to create regardless of the URL.
//	 If omitted, then it is derived from the serverURL scheme.
//	clientCert is optional to use certificate authentication, when supported.
//
// # Use SetTimeout for increasing the default communication timeout for testing
//
// This is intended to be used as a sink for application modules.
func NewTransportClient(protocolType string, serverURL string, clientCert *tls.Certificate, caCert *x509.Certificate,
	ch transports.ConnectionHandler) (cl transports.ITransportClient, err error) {

	parts, err := url.Parse(serverURL)
	scheme := strings.ToLower(parts.Scheme)
	// the protocol to use is based on scheme

	// use the URL to determine the protocol
	if protocolType == "" {
		if strings.HasPrefix(serverURL, transports.UriSchemeHiveotGrpc) {
			protocolType = transports.ProtocolTypeHiveotGrpc
		} else if strings.HasPrefix(serverURL, transports.UriSchemeWotWebsocket) {
			protocolType = transports.ProtocolTypeWotWebsocket
		} else if strings.HasPrefix(serverURL, transports.UriSchemeWotSse) {
			protocolType = transports.ProtocolTypeWotSse
		} else if strings.HasPrefix(serverURL, transports.UriSchemeWotMqtt) {
			protocolType = transports.ProtocolTypeWotMqtt
		} else if strings.HasPrefix(serverURL, transports.UriSchemeWotHttpBasic) {
			protocolType = transports.ProtocolTypeWotHttpBasic
		} else if strings.HasPrefix(serverURL, transports.UriSchemeHiveotSseSc) {
			protocolType = transports.ProtocolTypeHiveotSsesc
		}
	}

	switch protocolType {
	case transports.ProtocolTypeHiveotGrpc:
		// don't use TLS on unix domain sockets
		// if strings.HasPrefix(serverURL, "unix") {
		// 	caCert = nil
		// }
		cl = grpctransportpkg.NewHiveotGrpcClient(serverURL, clientCert, caCert, ch)

	case transports.ProtocolTypeHiveotSsesc:
		cl = ssescpkg.NewSseScClient(serverURL, clientCert, caCert, ch)

	case transports.ProtocolTypeHiveotWebsocket:
		cl = wsspkg.NewHiveotWssClient(serverURL, clientCert, caCert, ch)

	case transports.ProtocolTypeWotWebsocket:
		cl = wsspkg.NewWotWssClient(serverURL, clientCert, caCert, ch)

	case transports.ProtocolTypeWotHttpBasic:
		caCert := caCert
		cl = httpbasicpkg.NewHttpBasicClient(serverURL, clientCert, caCert, nil, ch)

	//case transports.ProtocolTypeWotMQTTWSS:
	//	fullURL = testServerMqttWssURL
	default:
		err = fmt.Errorf("NewTransportClient. Unknown protocol '%s'", scheme)
	}
	return cl, err
}

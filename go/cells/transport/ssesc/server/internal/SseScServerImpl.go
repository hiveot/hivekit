package internal

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/vocab"
	"github.com/hiveot/hivekit/go/api/wot"
	"github.com/hiveot/hivekit/go/cells/transport"
	"github.com/hiveot/hivekit/go/cells/transport/ssesc"
	"github.com/teris-io/shortid"
)

// SseScServerImpl is a transport for serving the HiveOT SSE-SC transport protocol.
// This implements the ITransportServer (and IHiveCell) interface.
//
// This transport protocol is build on top of HTTP and is bi-directional.
// It supports subscribing to events or observing properties.
type SseScServerImpl struct {
	// Transport base includes the RnR channel for matching request-response messages.
	*transport.TransportServerBase

	// flag, include affordances when adding forms
	includeAffordances bool

	// SSE-Sc protocol message encoder
	encoder transport.IMessageEncoder

	// actual server exposing routes
	httpServer api.IHttpServer

	// The SSE connection path
	ssePath string

	// serverTD is the TD describing how to connect to this server
	serverTD *wot.TD
}

// AddTDSecForms updates the TD with base URI, security scheme and forms for use of
// this protocol to the given TD.
//
// Since the contentType is the default application/json it is omitted
func (srv *SseScServerImpl) AddTDSecForms(tdoc *wot.TD) {

	// 1. Add the base connection endpoint
	// if this Thing supports multiple protocols the first one occupies 'base'.
	// Additional protocols will require a full href in their forms.
	base := srv.GetConnectURL()
	if tdoc.Base == "" {
		tdoc.Base = base
		base = "" // no need for full base
	}
	vars := map[string]string{
		wot.UriVarThingID: tdoc.ID,
	}
	// protocolType := api.ProtocolTypeHiveotSsesc
	subprotocol := api.HiveotSsescSubprotocol

	// 2. Set the security scheme used by the authenticator.
	authr := srv.httpServer.GetAuthenticator()
	authr.AddSecurityScheme(tdoc)

	// 3. add thing level form for thing level operations
	// since the payload is a request message, one relative path for all request operations
	href2 := base + ssesc.PostSseScRequestPath
	form := wot.NewForm("", href2)
	form.SetSubprotocol(subprotocol)
	form["op"] = []string{
		wot.HTOpPing,
		wot.OpQueryAllActions,
		wot.OpObserveAllProperties, wot.OpUnobserveAllProperties,
		wot.OpReadAllProperties,
		wot.HTOpReadAllEvents, // hiveot supports reading latest events
		wot.OpSubscribeAllEvents, wot.OpUnsubscribeAllEvents,
	}
	tdoc.Forms = append(tdoc.Forms, form)

	// 4. Add forms to all affordances to be compliant with the specifications.
	// This does uses the same href to prevent conflict with multiple protocols
	if srv.includeAffordances {

		for _, aff := range tdoc.Actions {
			form := aff.AddForm("", href2, http.MethodPost, vars)
			form.SetSubprotocol(subprotocol)
			form["op"] = []string{wot.OpInvokeAction, wot.OpQueryAction}
		}
		for _, aff := range tdoc.Events {
			form := aff.AddForm("", href2, "", nil)
			form.SetSubprotocol(subprotocol)
			form["op"] = []string{wot.HTOpReadEvent, wot.OpSubscribeEvent, wot.OpUnsubscribeEvent}
		}
		for _, aff := range tdoc.Properties {
			form := aff.AddForm("", href2, "", nil)
			form.SetSubprotocol(subprotocol)
			if !aff.WriteOnly {
				form["op"] = []string{wot.OpReadProperty, wot.OpObserveProperty, wot.OpUnobserveProperty}
			}
			if !aff.ReadOnly {
				form["op"] = []string{wot.OpWriteProperty}
			}
			form["op"] = []string{wot.HTOpReadEvent, wot.OpSubscribeEvent, wot.OpUnsubscribeEvent}
		}
	}
}

// GetTD returns the server TD, containing connection and authentication information
func (srv *SseScServerImpl) GetTD() *wot.TD {
	return srv.serverTD
}

// Stop any running actions
func (srv *SseScServerImpl) Stop() {
	slog.Info("Stop: Stopping ssesc transport server")
	srv.CloseAll()
}

// Start a new HiveOT Http/SSE server using the given http server.
// The http server must have authentication setup
//
// # The optional connect handler is invoked when connections appear and disappear
//
// Use SetRequestSink to set the handler for requests send by consumers
// Use SetNotificationSink to set the handler for notifications send by Things.
func StartSseScServerImpl(httpServer api.IHttpServer) (*SseScServerImpl, error) {

	ssePath := ssesc.SseScPath
	slog.Info("Start: Starting ssesc transport server", "ssePath", ssePath)

	httpAddr := httpServer.GetConnectURL()
	urlParts, _ := url.Parse(httpAddr)

	connectURL := fmt.Sprintf("%s://%s%s", api.HiveotSseScScheme, urlParts.Host, ssePath)

	// use the RRN message format. Simple passthrough.
	encoder := transport.NewRRNJsonEncoder()

	thingID := ssesc.SseScServerCellType + "-" + shortid.MustGenerate()
	authenticator := httpServer.GetAuthenticator()
	serverTD := wot.NewTD(thingID, "SSE-SC server", vocab.DeviceTypeService)

	srv := &SseScServerImpl{
		TransportServerBase: transport.NewTransportServerBase(thingID, connectURL, authenticator),
		httpServer:          httpServer,
		ssePath:             ssePath,
		encoder:             encoder,
		serverTD:            serverTD,
	}
	// Add the routes used in SSE connection and subscription requests
	srv.CreateRoutes(srv.ssePath, srv.httpServer.GetProtectedRoute())

	// create a TD describing this server along with its connection URL
	srv.AddTDSecForms(srv.serverTD)

	var _ api.IHiveCell = srv        // interface check
	var _ api.ITransportServer = srv // interface check

	return srv, nil
}

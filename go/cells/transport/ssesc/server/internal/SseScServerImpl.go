package internal

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/api/vocab"
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

	// waiting for response timeout (see rnr)
	respTimeout time.Duration

	// The SSE connection path
	ssePath string

	// serverTD is the TD describing how to connect to this server
	serverTD *td.TD
}

// AddTDSecForms updates the TD with base URI, security scheme and forms for use of
// this protocol to the given TD.
//
// Since the contentType is the default application/json it is omitted
func (srv *SseScServerImpl) AddTDSecForms(tdoc *td.TD) {
	// 1. Add the base connection endpoint
	// TODO: if this Thing supports multiple protocols it might conflict with
	// the base. In that case base cannot be used and all hrefs must be absolute?
	href := srv.GetConnectURL()
	tdoc.Base = href
	vars := map[string]string{
		td.UriVarThingID: tdoc.ID,
	}
	// protocolType := api.ProtocolTypeHiveotSsesc
	subprotocol := api.HiveotSsescSubprotocol

	// 2. Set the security scheme used by the authenticator.
	// TODO: risk of duplicates?
	authr := srv.httpServer.GetAuthenticator()
	authr.AddSecurityScheme(tdoc)

	// 3. add thing level form for thing level operations
	// since the payload is a request message, one relative path for all request operations
	href2 := ssesc.PostSseScRequestPath
	form := td.NewForm("", href2)
	form.SetSubprotocol(subprotocol)
	form["op"] = []string{
		td.HTOpPing,
		td.OpQueryAllActions,
		td.OpObserveAllProperties, td.OpUnobserveAllProperties,
		td.OpReadAllProperties,
		td.HTOpReadAllEvents, // hiveot supports reading latest events
		td.OpSubscribeAllEvents, td.OpUnsubscribeAllEvents,
	}
	tdoc.Forms = append(tdoc.Forms, form)

	// 4. Add forms to all affordances to be compliant with the specifications.
	// This does uses the same href to prevent conflict with multiple protocols
	if srv.includeAffordances {

		for _, aff := range tdoc.Actions {
			form := aff.AddForm("", href2, http.MethodPost, vars)
			form.SetSubprotocol(subprotocol)
			form["op"] = []string{td.OpInvokeAction, td.OpQueryAction}
		}
		for _, aff := range tdoc.Events {
			// todo subscribe to events by connecting to endpoint
			form := aff.AddForm("", href2, "", nil)
			form.SetSubprotocol(subprotocol)
			form["op"] = []string{td.HTOpReadEvent, td.OpSubscribeEvent, td.OpUnsubscribeEvent}
		}
		for _, aff := range tdoc.Properties {
			// todo subscribe to props by connecting to endpoint
			form := aff.AddForm("", href2, "", nil)
			form.SetSubprotocol(subprotocol)
			if !aff.WriteOnly {
				form["op"] = []string{td.OpReadProperty, td.OpObserveProperty, td.OpUnobserveProperty}
			}
			if !aff.ReadOnly {
				form["op"] = []string{td.OpWriteProperty}
			}
			form["op"] = []string{td.HTOpReadEvent, td.OpSubscribeEvent, td.OpUnsubscribeEvent}
		}
	}
}

// GetTD returns the server TD, containing connection and authentication information
func (srv *SseScServerImpl) GetTD() *td.TD {
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
func StartSseScServerImpl(
	httpServer api.IHttpServer, respTimeout time.Duration) (*SseScServerImpl, error) {

	ssePath := ssesc.SseScPath
	slog.Info("Start: Starting ssesc transport server", "ssePath", ssePath)

	httpAddr := httpServer.GetConnectURL()
	urlParts, _ := url.Parse(httpAddr)

	connectURL := fmt.Sprintf("%s://%s%s", api.HiveotSseScScheme, urlParts.Host, ssePath)

	// use the RRN message format. Simple passthrough.
	encoder := transport.NewRRNJsonEncoder()
	if respTimeout == 0 {
		respTimeout = msg.DefaultRnRTimeout
	}

	thingID := ssesc.SseScServerCellType + "-" + shortid.MustGenerate()
	authenticator := httpServer.GetAuthenticator()
	serverTD := td.NewTD(thingID, "SSE-SC server", vocab.DeviceTypeService)

	srv := &SseScServerImpl{
		TransportServerBase: transport.NewTransportServerBase(thingID, connectURL, authenticator),
		httpServer:          httpServer,
		ssePath:             ssePath,
		encoder:             encoder,
		respTimeout:         respTimeout,
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

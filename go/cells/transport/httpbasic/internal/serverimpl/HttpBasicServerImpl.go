package serverimpl

import (
	"fmt"
	"log/slog"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/api/vocab"
	"github.com/hiveot/hivekit/go/cells/transport"
	"github.com/hiveot/hivekit/go/cells/transport/httpbasic"
	"github.com/teris-io/shortid"
)

// HTTP-basic profile constants
const (
	// static file server routes
	DefaultHttpStaticBase      = "/static"
	DefaultHttpStaticDirectory = "stores/httpstatic" // relative to home
)

// HttpBasicServerImpl is a transport server for serving the wot http-basic protocol.
// This implements the ITransportServer and IHttpServer interfaces.
//
// This WoT defined protocol is build on top of HTTP and is uni-directional.
// It is only intended for consumers and not for exposed things using connection reversal.
// It does not support subscribing to events or observing properties.
type HttpBasicServerImpl struct {
	*transport.TransportServerBase

	// actual httpServer exposing routes
	httpServer api.IHttpServer

	// the TD describing this server
	serverTD *td.TD
}

func (srv *HttpBasicServerImpl) GetHttpServer() api.IHttpServer {
	return srv.httpServer
}

// GetTD returns the server TD, containing connection and authentication information
func (srv *HttpBasicServerImpl) GetTD() *td.TD {
	return srv.serverTD
}

// HandleRequest passes request messages to the API handler.
// If the request isn't for this cell then this returns an error as the server
// cannot deliver messages to the client.
func (srv *HttpBasicServerImpl) HandleRequest(
	req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {

	err = fmt.Errorf("SendRequest. HTTP can't send requests to remote clients.")
	slog.Error(err.Error())
	return err
}

// Stop any running actions
func (srv *HttpBasicServerImpl) Stop() {
	slog.Info("Stop: Stopping httpbasic transport server")

}

// StartHttpBasicServerImpl starts a new WoT http-basic protocol binding and starts listening.
//
// This:
// - adds a public routes for ping
// - adds a protected route for thing requests {op}/{thing}/{name}
// - adds a protected route for affordance requests {op}/{thing}/{affordance}/{name}
//
// Incoming requests are converted to the standard message format and passed to
// the registered sinks.
//
// This supports the HandleRequest - read(all)properties RRN to retrieve statistics
// of the http transport.
//
// Since http is a unidirectional protocol, HandleNotification and HandleRequest messages
// will not be passed to connected clients.
//
// Intended as a last-resort server as this only handles consumer connections and
// does not support subscription.
// The onRequest handler only handles responses that are sent via replyTo in a short
// timeframe. (eg timeout setting)
//
//	httpServer is the http server that listens for messages
//	sink is the optional receiver of request, response and notification messages, nil to set later
func StartHttpBasicServerImpl(httpServer api.IHttpServer) (*HttpBasicServerImpl, error) {

	slog.Info("Start: Starting httpbasic transport server")

	thingID := httpbasic.HttpBasicServerCellType + "-" + shortid.MustGenerate()
	connectURL := httpServer.GetConnectURL()
	authenticator := httpServer.GetAuthenticator()
	srv := &HttpBasicServerImpl{
		TransportServerBase: transport.NewTransportServerBase(thingID, connectURL, authenticator),
		httpServer:          httpServer,
	}
	// create a TD describing this server along with its connection URL
	srv.serverTD = td.NewTD(thingID, "HTTP-Basic server", vocab.DeviceTypeService)
	srv.AddTDSecForms(srv.serverTD, false)

	err := srv.createRoutes()

	var _ api.ITransportServer = srv // interface check
	var _ api.IHiveCell = srv        // interface check

	return srv, err
}

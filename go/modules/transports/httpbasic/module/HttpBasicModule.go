package module

import (
	"fmt"
	"log/slog"

	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/modules/transports/httpbasic"
	"github.com/hiveot/hivekit/go/modules/transports/httpbasic/api"
	"github.com/hiveot/hivekit/go/modules/transports/httpbasic/service"
	"github.com/hiveot/hivekit/go/modules/transports/httpserver"
	"github.com/hiveot/hivekit/go/msg"
)

// NewHttpBasicModule is a module for serving the wot http-basic protocol.
// This implements IHiveModule interface.
//
// The module's HandleRequest and HandleNotification methods can be used to
// send messages to connected clients.
// Handlers of received messages can be added as sinks to this module.
// Since http is a connectionless protocol, this does not have the onConnect hook
// that other transports have.
type HttpBasicModule struct {
	modules.HiveModuleBase

	// the RRN messaging receiver
	rrnAPI *api.HttpBasicRRNHandler

	// actual server exposing routes
	server httpserver.IHttpServer

	// the linked authenticator
	authenticator transports.IAuthenticator

	// http-basic protocol server
	service *service.HttBasicServer
}

// HandleRequest passes the module request messages to the API handler.
func (m *HttpBasicModule) HandleRequest(req *msg.RequestMessage) (resp *msg.ResponseMessage) {
	if m.rrnAPI != nil {
		resp = m.rrnAPI.HandleRequest(req)
	}
	// the module base handles operations for reading properties
	if resp == nil {
		resp = m.HiveModuleBase.HandleRequest(req)
	}
	return resp
}

func (m *HttpBasicModule) onNotificationMessage(notif *msg.NotificationMessage) {
	// Agent client has sent a notification. Forward to the sinks.
	m.SendNotification(notif)
}
func (m *HttpBasicModule) onRequestMessage(req *msg.RequestMessage, sender transports.IConnection) (resp *msg.ResponseMessage) {
	// FIXME: the pipeline doesn't support async response messages
	// option 1: add it
	// option 2: remove support for async responses. Instead wait for a response during send.
	return m.SendRequest(req)
}

func (m *HttpBasicModule) onResponseMessage(resp *msg.ResponseMessage) (err error) {
	// Two issues here to be fixed
	// 1. support async response messages, send by agents
	// 2. oops, forgot
	err = fmt.Errorf("onResponseMessage: FIXME: receiving response message not supported")
	slog.Error(err.Error())
	return err
}

// Start readies the module for use.
// This:
// - sets-up a middleware chain with recovery, compression, authentication
// - create a protected and public route that can be used to others
// Configurable:
// - add public routes for login and ping
// - add protected route for thing requests {operation}/{thing}/{name}
// - add protected route for affordance requests {operation}/{thing}/{affordance}/{name}
// - add protected routes for token refresh and logout
//
// Incoming requests are converted to the standard message format and passed to
// the registered sinks.
//
// This supports the HandleRequest - read(all)properties RRN to retrieve statistics
// of the http transport.
//
// Since http is a unidirectional protocol, HandleNotification and HandleRequest messages
// will not be passed to connected clients.
func (m *HttpBasicModule) Start() (err error) {

	m.service = service.NewHttpBasicServer(m.server,
		// m.router, m.authenticator,
		// received messages are passed to the sinks
		m.onNotificationMessage, m.onRequestMessage, m.onResponseMessage)

	err = m.service.Start()

	// the basic msg handler handles incoming messages. Messages addressed
	// to established connections are passed to the client. Messages addressed
	// to this module are processed directly.
	// all remaining messages are passed to the sinks.
	if err == nil {
		m.rrnAPI = api.NewHttpBasicMsgHandler(m.ModuleID, m.service)
	}
	return err
}

// Stop any running actions
func (m *HttpBasicModule) Stop() {
	m.service.Stop()
}

// Start a new WoT HTTP-Basic server using the given public/protected router.
func NewHttpBasicModule(server httpserver.IHttpServer, authenticator transports.IAuthenticator) *HttpBasicModule {

	m := &HttpBasicModule{
		HiveModuleBase: modules.HiveModuleBase{
			ModuleID:   httpbasic.DefaultHttpBasicThingID,
			Properties: make(map[string]any),
		},
		server:        server,
		authenticator: authenticator,
	}
	var _ modules.IHiveModule = m // interface check

	return m
}

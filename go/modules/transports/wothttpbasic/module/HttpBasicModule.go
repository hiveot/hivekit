package module

import (
	"fmt"
	"log/slog"

	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/messaging"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/modules/transports/wothttpbasic"
	"github.com/hiveot/hivekit/go/modules/transports/wothttpbasic/api"
	"github.com/hiveot/hivekit/go/modules/transports/wothttpbasic/service"
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

	// the SME messaging API
	msgAPI *api.HttpBasicMsgHandler

	// router for rest api
	router *chi.Mux

	// the linked authenticator
	authenticator transports.IAuthenticator

	// http-basic protocol server
	service *service.HttBasicServer
}

// HandleRequest passes the module request messages to the API handler.
func (m *HttpBasicModule) HandleRequest(req *messaging.RequestMessage) (resp *messaging.ResponseMessage) {
	if m.msgAPI != nil {
		resp = m.msgAPI.HandleRequest(req)
	}
	// the module base handles operations for reading properties
	if resp == nil {
		resp = m.HiveModuleBase.HandleRequest(req)
	}
	return resp
}

func (m *HttpBasicModule) onNotificationMessage(notif *messaging.NotificationMessage) {
	// Agent client has sent a notification. Forward to the sinks.
	m.SendNotification(notif)
}
func (m *HttpBasicModule) onRequestMessage(req *messaging.RequestMessage, sender transports.IConnection) (resp *messaging.ResponseMessage) {
	// FIXME: the pipeline doesn't support async response messages
	// option 1: add it
	// option 2: remove support for async responses. Instead wait for a response during send.
	return m.SendRequest(req)
}

func (m *HttpBasicModule) onResponseMessage(resp *messaging.ResponseMessage) (err error) {
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
// This supports the HandleRequest - read(all)properties SME to retrieve statistics
// of the http transport.
//
// Since http is a unidirectional protocol, HandleNotification and HandleRequest messages
// will not be passed to connected clients.
func (m *HttpBasicModule) Start() (err error) {

	addr := "" // ?
	m.service = service.NewHttpBasicServer(
		addr,
		m.router, m.authenticator,
		// received messages are passed to the sinks
		m.onNotificationMessage, m.onRequestMessage, m.onResponseMessage)

	err = m.service.Start()

	// the basic msg handler handles incoming messages. Messages addressed
	// to established connections are passed to the client. Messages addressed
	// to this module are processed directly.
	// all remaining messages are passed to the sinks.
	if err == nil {
		m.msgAPI = api.NewHttpBasicMsgHandler(m.ModuleID, m.service)
	}
	return err
}

// Stop any running actions
func (m *HttpBasicModule) Stop() {
	m.service.Stop()
}

// Start a new WoT HTTP-Basic server using the given router and authenticator.
//
// This can work with any HTTPS server that supports the chi router.
//
// router is the html server router to register the html API handlers with.
func NewHttpBasicModule(router *chi.Mux, authenticator transports.IAuthenticator) *HttpBasicModule {

	m := &HttpBasicModule{
		HiveModuleBase: modules.HiveModuleBase{
			ModuleID:   wothttpbasic.DefaultHttpBasicThingID,
			Properties: make(map[string]any),
		},
		authenticator: authenticator,
		router:        router,
	}
	return m
}

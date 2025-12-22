package module

import (
	"crypto/tls"
	"crypto/x509"

	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/transports/httpserver"
	"github.com/hiveot/hivekit/go/modules/transports/httpserver/service"
	"github.com/hiveot/hivekit/go/msg"
)

// HttpServerModule is a module providing a TLS HTTPS server.
// Intended for use by HTTP based application protocols.
// This implements IHiveModule interface.
type HttpServerModule struct {
	modules.HiveModuleBase

	// certificate handler for running the server
	caCert     *x509.Certificate
	serverCert *tls.Certificate

	config *httpserver.HttpServerConfig

	// The router available for this TLS server
	// Intended for Http modules to add their routes
	router *chi.Mux

	// the RRN messaging API
	// msgAPI *api.HttpMsgHandler

	// TLS protocol server
	service *service.HttpsServer
}

func (m *HttpServerModule) GetService() *service.HttpsServer {
	return m.service
}

// HandleRequest passes the module RRN request messages to the message handler.
// currently this module does not expose properties or actions to request.
func (m *HttpServerModule) HandleRequest(req *msg.RequestMessage) (resp *msg.ResponseMessage) {
	// if m.msgAPI != nil {
	// 	resp = m.msgAPI.HandleRequest(req)
	// }
	// the module base handles operations for reading properties
	if resp == nil {
		resp = m.HiveModuleBase.HandleRequest(req)
	}
	return resp
}

// // onNotificationMessage service generated a notification
// func (m *TlsModule) onNotificationMessage(notif *msg.NotificationMessage) {
// 	m.SendNotification(notif)
// }

// // onRequestMessage service generated a request message
// func (m *TlsModule) onRequestMessage(req *msg.RequestMessage, sender transports.IConnection) (resp *msg.ResponseMessage) {
// 	// FIXME: the pipeline doesn't support async response messages
// 	// option 1: add it
// 	// option 2: remove support for async responses. Instead wait for a response during send.
// 	return m.SendRequest(req)
// }

// // onResponseMessage service generated a response message
// func (m *TlsModule) onResponseMessage(resp *msg.ResponseMessage) (err error) {
// 	// Two issues here to be fixed
// 	// 1. support async response messages, send by agents
// 	// 2. oops, forgot
// 	err = fmt.Errorf("onResponseMessage: FIXME: receiving response message not supported")
// 	slog.Error(err.Error())
// 	return err
// }

// Start readies the module for use.
//
// Starts a HTTPS TLS service
func (m *HttpServerModule) Start() (err error) {
	m.service = service.NewHttpsServer(m.config)
	err = m.service.Start()
	return err
}

// Stop any running actions
func (m *HttpServerModule) Stop() {
	m.service.Stop()
}

// Create a new Https server module instance.
//
// moduleID is the module's instance identification.
// config MUST have been configured with a CA and server certificate unless
// NoTLS is set.
func NewHttpServerModule(moduleID string, config *httpserver.HttpServerConfig) *HttpServerModule {

	if moduleID == "" {
		moduleID = httpserver.DefaultHttpServerModuleID
	}

	m := &HttpServerModule{
		HiveModuleBase: modules.HiveModuleBase{
			ModuleID:   moduleID,
			Properties: make(map[string]any),
		},
		config: config,
	}
	var _ modules.IHiveModule = m // interface check
	return m
}

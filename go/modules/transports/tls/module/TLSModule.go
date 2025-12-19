package module

import (
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/messaging"
	"github.com/hiveot/hivekit/go/modules/services/certs"
	"github.com/hiveot/hivekit/go/modules/transports/tls"
	"github.com/hiveot/hivekit/go/modules/transports/tls/service"
)

// TlsModule is a module for serving the TLS HTTPS server.
// This implements IHiveModule interface.
type TLSModule struct {
	modules.HiveModuleBase

	// certificate handler for running the server
	certs certs.ICertsService

	// The listening address or "" for all available addresses
	addr string
	// The listening port or 444 when not set
	port int

	// The router available for this TLS server
	// Intended for Http modules to add their routes
	router *chi.Mux

	// the SME messaging API
	// msgAPI *api.TLSMsgHandler

	// TLS protocol server
	service *service.TLSServer
}

func (m *TLSModule) GetService() tls.ITLSTransport {
	return m.service
}

// HandleRequest passes the module SME request messages to the message handler.
func (m *TLSModule) HandleRequest(req *messaging.RequestMessage) (resp *messaging.ResponseMessage) {
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
// func (m *TlsModule) onNotificationMessage(notif *messaging.NotificationMessage) {
// 	m.SendNotification(notif)
// }

// // onRequestMessage service generated a request message
// func (m *TlsModule) onRequestMessage(req *messaging.RequestMessage, sender transports.IConnection) (resp *messaging.ResponseMessage) {
// 	// FIXME: the pipeline doesn't support async response messages
// 	// option 1: add it
// 	// option 2: remove support for async responses. Instead wait for a response during send.
// 	return m.SendRequest(req)
// }

// // onResponseMessage service generated a response message
// func (m *TlsModule) onResponseMessage(resp *messaging.ResponseMessage) (err error) {
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
func (m *TLSModule) Start() (err error) {

	m.router = chi.NewRouter()
	caCert := m.certs.GetCACert()
	serverTlsCert := m.certs.GetServerCert(m.ModuleID)

	m.service, m.router = service.NewTLSServer(
		m.addr, m.port,
		serverTlsCert,
		caCert,
		m.router)

	err = m.service.Start()

	return err
}

// Stop any running actions
func (m *TLSModule) Stop() {
	m.service.Stop()
}

// Start a new TLS server
//
// This can work with any HTTPS server that supports the chi router.
func NewTLSModule(addr string, port int, certs certs.ICertsService) *TLSModule {

	m := &TLSModule{
		HiveModuleBase: modules.HiveModuleBase{
			ModuleID:   tls.DefaultTlsThingID,
			Properties: make(map[string]any),
		},
		addr:  addr,
		port:  port,
		certs: certs,
	}
	var _ modules.IHiveModule = m // interface check
	return m
}

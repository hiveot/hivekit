package module

import (
	"fmt"
	"log/slog"
	"net/url"

	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/modules/transports/direct"
	"github.com/hiveot/hivekit/go/modules/transports/httptransport"
	"github.com/hiveot/hivekit/go/modules/transports/wss"
	wssapi "github.com/hiveot/hivekit/go/modules/transports/wss/api"
	"github.com/hiveot/hivekit/go/msg"
)

// WssModule is a transport module that serves Websocket connections over http.
type WssModule struct {
	transports.TransportModuleBase
	// this handles request for this module
	msgAPI *wssapi.WssMsgAPI

	// actual server exposing routes including websocket endpoint
	httpServer httptransport.IHttpServer

	// Websocket protocol message converter
	msgConverter transports.IMessageConverter // WoT or Hiveot message format

	// registered handler to notify of incoming connections
	serverConnectHandler transports.ConnectionHandler

	//
	subprotocol string // WoT or Hiveot subprotocol
	// listening path for incoming connections
	wssPath string
}

// HandleRequest passes the module request messages to the API handler.
// This has nothing to do with receiving requests over websockets.
func (m *WssModule) HandleRequest(
	req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {

	// first attempt to procss the when targeted at this module
	if req.ThingID == m.GetModuleID() {
		err = m.msgAPI.HandleRequest(req, replyTo)
	}
	// if the request failed, then forward the request through the chain
	// the module base handles operations for reading properties
	if err != nil {
		err = m.HiveModuleBase.HandleRequest(req, replyTo)
	}
	return err
}

// Start listening for incoming websocket connections
func (m *WssModule) Start() (err error) {
	slog.Info("Starting websocket module, Listening on: " + m.GetConnectURL())
	// TODO: detect if already listening
	err = m.TransportModuleBase.Start()
	// create routes
	router := m.httpServer.GetProtectedRoute()
	router.Get(m.wssPath, m.Serve)

	// The basic msg handler converts incoming module requests messages to the module API.
	// This has nothing to do with the http server.
	if err == nil {
		m.msgAPI = wssapi.NewWssMsgAPI(m)
	}
	return nil
}

// Stop disconnects clients and remove connection listening
func (m *WssModule) Stop() {
	slog.Info("Stopping websocket module")
	m.CloseAll()
	router := m.httpServer.GetProtectedRoute()
	router.Delete(m.wssPath, m.Serve)
}

// Create a websocket module using hiveot RRN direct messaging
// This is used for agents that use reverse connections.
// This uses a passthrough message converter for requests, response and notifications
//
// httpServer is the http server the websocket is using
// sink is the optional receiver of request, response and notification messages, nil to set later
func NewHiveotWssModule(httpServer httptransport.IHttpServer, sink modules.IHiveModule) *WssModule {

	httpURL := httpServer.GetConnectURL()
	urlParts, err := url.Parse(httpURL)
	if err != nil {
		panic("NewHiveotWssModule: Http server has invalid URL")
	}

	m := &WssModule{
		httpServer:           httpServer,
		msgConverter:         direct.NewPassthroughMessageConverter(),
		subprotocol:          wss.SubprotocolHiveotWSS,
		serverConnectHandler: nil,
		wssPath:              wss.DefaultHiveotWssPath,
	}
	moduleID := wss.DefaultHiveotWssModuleID
	connectURL := fmt.Sprintf("%s://%s%s", wss.HiveotWssSchema, urlParts.Host, m.wssPath)
	// set the base parameters
	m.Init(moduleID, sink, connectURL, transports.DefaultRpcTimeout)
	return m
}

// Create a websocket module using WoT messaging format
// This uses the WoT websocket protocol message converter to convert between
// the standard RRN messages and the WoT websocket message format.
//
// httpServer is the http server the websocket is using
// sink is the optional receiver of request, response and notification messages, nil to set later
func NewWotWssModule(httpServer httptransport.IHttpServer, sink modules.IHiveModule) *WssModule {
	httpURL := httpServer.GetConnectURL()
	urlParts, err := url.Parse(httpURL)
	if err != nil {
		panic("NewWotWssModule: Http server has invalid URL")
	}
	m := &WssModule{
		httpServer:   httpServer,
		msgConverter: wssapi.NewWotWssMsgConverter(),
		subprotocol:  wss.SubprotocolWotWSS,
		wssPath:      wss.DefaultWotWssPath,
	}
	moduleID := wss.DefaultWotWssModuleID
	connectURL := fmt.Sprintf("%s://%s%s", wss.WotWssSchema, urlParts.Host, m.wssPath)

	m.Init(moduleID, sink, connectURL, transports.DefaultRpcTimeout)
	return m
}

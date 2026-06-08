package transport

import (
	"time"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules"
)

// Transport server module definitions for use by transport protocols.
// This contains the protocol types, authentication handler, and interfaces for the shared http server and tls client.

// notifications sent by transport servers to server side services
// These are published by TransportServerBase
const (
	// The server accepted a connection from a client
	ServerConnectEvent = "serverconnect"
	// The server remove a client connection
	ServerDisconnectEvent = "serverdisconnect"
)

const (
	// HiveOT SSE uses a single SSE connection as return channel; payload are RRN messages.
	ProtocolTypeHiveotSsesc   = "hiveot-ssesc"
	ProtocolSchemeHiveotSseSc = "sse"
	SubprotocolHiveotSsesc    = "sse-sc"

	// HiveOT gRPC is intended for local inter-process communication using UDS,
	// and uses the HiveOT RRN messages as the payload.
	// TODO: also support the tcp variant
	ProtocolTypeHiveotGrpc   = "hiveot-grpc"
	ProtocolSchemeHiveotGrpc = "unix"
	SubprotocolHiveotGrpc    = "" // not a subprotocol

	// HiveOT websocket uses RRN messages as the envelope.
	ProtocolTypeHiveotWebsocket   = "hiveot-websocket"
	ProtocolSchemeHiveotWebsocket = "wss"
	SubprotocolHiveotWebsocket    = "hiveot:websocket"

	// Http-basic follows the WoT specification
	ProtocolTypeWotHttpBasic   = "http-basic"
	ProtocolSchemeWotHttpBasic = "https"
	SubprotocolWotHttpBasic    = ""

	// Http long poll is not implemented
	ProtocolTypeWotHttpLongPoll   = "http-longpoll"
	ProtocolSchemeWotHttpLongPoll = "https"
	SubprotocolWotHttpLongPoll    = "longpoll"

	// WoT MQTT is not yet implemented
	ProtocolTypeWotMqtt   = "wot-mqtt"
	ProtocolSchemeWotMqtt = "mqtts"

	// WoT SSE is not implemented
	ProtocolTypeWotSse   = "wot-sse"
	ProtocolSchemeWotSse = "sse"

	// WoT websocket follows the WoT specification
	ProtocolTypeWotWebsocket   = "wot-websocket"
	ProtocolSchemeWotWebsocket = "wss"
	SubprotocolWotWebsocket    = "websocket"
)

// payload of connection events
type ConnectionInfo struct {
	// ClientID holds the account ID of the connected client
	ClientID string `json:"clientID"`
	// ConnectionID holds the instance ID of the connected client
	ConnectionID string `json:"cid"`
}

// ValidateToken verifies the token and client are valid.
// This returns an error if the token is invalid, the token has expired,
// or the client is not a valid and enabled client.
type ValidateTokenHandler func(token string) (clientID string, validUntil time.Time, err error)

// A transport server module is a server module with hooks for sending messages to remote clients.
type ITransportServer interface {
	modules.IHiveModule

	// AddTDSecForms updates the given Thing Description with security and forms for this
	// transport module.
	// The security scheme in the TD is set by the authenticator used by the server.
	AddTDSecForms(tdoc *td.TD, includeAffordances bool)

	// CloseAll closes all client connections. Mainly intended for testing.
	CloseAll()

	// Return the established connection of the given client, if one exists
	// This returns nil if the client does not have an authenticated connection.
	GetConnectionByClientID(clientID string) IConnection

	// GetConnectURL returns connection URL of the server
	GetConnectURL() (uri string)

	// HandleNotification sends the notification to subscribed clients using SendNotification.
	// The remote clients are the notification sink from the server perspective.
	//
	// Notifications received by the server are forwarded to the notification sink
	HandleNotification(notif *msg.NotificationMessage)

	// SendNotification [agent] sends a notification over the connections to
	// remote subscribed consumers.
	SendNotification(notif *msg.NotificationMessage)

	// SendRequest [consumer] sends a request to a connected agent.
	//
	// Intended for use by consumers when agents are connected using connection reversal.
	//
	// agentID is the agent's authentication ID that hosts the Thing.
	// responseHandler is the optional callback with the response.
	//
	// This returns an error if the agent is no longer connected.
	SendRequest(agentID string, req *msg.RequestMessage, replyTo msg.ResponseHandler) error

	// SendResponse [agent] sends the response message over the transport to a remote
	// consumer with the given client and connection ID.
	//
	// Intended for use by agents that host one or more Things.
	SendResponse(clientID string, cid string, resp *msg.ResponseMessage) error
}

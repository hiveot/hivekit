package api

import (
	"time"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
)

// Transport server module definitions for use by transport protocols.
// This contains the protocol types, authentication handler, and interfaces for the shared http server and tls client.

// notifications sent by transport servers to server side services
// These are published by TransportServerBase
const (
	// The server accepted a connection from a client
	ServerConnectedEvent = "serverconnect"
	// The server remove a client connection
	ServerDisconnectedEvent = "serverdisconnect"
)

// Note: the definition of protocol-type is scheme:subprotocol
const (
	// HiveOT SSE uses a single SSE connection as return channel; payload are RRN messages.
	HiveotSseScScheme       = "sse"
	HiveotSsescSubprotocol  = "sse-sc"
	HiveotSseScProtocolType = HiveotSseScScheme + ":" + HiveotSsescSubprotocol

	// HiveOT gRPC is intended for local inter-process communication using UDS,
	// and uses the HiveOT RRN messages as the payload.
	// TODO: also support the tcp variant
	HiveotGrpcUnixScheme       = "unix"
	HiveotGrpcUnixSubprotocol  = "hiveot-grpc"
	HiveotGrpcUnixProtocolType = HiveotGrpcUnixScheme + ":" + HiveotGrpcUnixSubprotocol

	HiveotGrpcTcpScheme       = "tcp"
	HiveotGrpcTcpSubprotocol  = "hiveot-grpc"
	HiveotGrpcTcpProtocolType = HiveotGrpcTcpScheme + ":" + HiveotGrpcTcpSubprotocol

	// HiveOT websocket uses RRN messages as the envelope.
	HiveotWebsocketScheme       = "wss"
	HiveotWebsocketSubprotocol  = "hiveot:websocket"
	HiveotWebsocketProtocolType = HiveotWebsocketScheme + ":" + HiveotWebsocketSubprotocol

	// Http-basic follows the WoT specification
	HttpBasicScheme       = "https"
	HttpBasicSubprotocol  = ""
	HttpBasicProtocolType = HttpBasicScheme + ":" + HttpBasicSubprotocol

	// WoT MQTT is not yet implemented
	WotMqttsScheme      = "mqtts"
	WotMqttsSubprotocol = ""
	WotMqttProtocolType = WotMqttsScheme + ":" + WotMqttsSubprotocol

	WotMqttWebsocketScheme       = "wss"
	WotMqttWebsocketSubprotocol  = "mqtt" // need a subprotocol to differentiate other websockets?
	WotMqttWebsocketProtocolType = WotMqttWebsocketScheme + ":" + WotMqttWebsocketSubprotocol

	// WoT SSE is not implemented
	// ProtocolTypeWotSse   = "wot-sse"
	// ProtocolSchemeWotSse = "sse"

	// WoT websocket follows the WoT specification
	WotWebsocketScheme       = "wss"
	WotWebsocketSubprotocol  = "websocket"
	WotWebsocketProtocolType = WotWebsocketScheme + ":" + WotWebsocketSubprotocol
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
	IHiveModule

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

	// Return the server's TD.
	// This TD describes the server itself and provides a base URL for its connection
	// Primarily intended for testing. In most cases a server is run together with a
	// device module whose TD is updated with forms from the server.
	GetTD() *td.TD

	// HandleNotification sends the notification to subscribed clients using SendNotification.
	// The remote clients are the notification sink from the server perspective.
	//
	// Notifications received by the server are forwarded to the notification sink
	// This returns an error if the notification is not handled or nil if at least one
	// client subscribes.
	HandleNotification(notif *msg.NotificationMessage)

	// SendNotification [Thing] sends a notification over the connections to
	// remote subscribed consumers.
	// This returns an error if the notification has no subscribers.
	SendNotification(notif *msg.NotificationMessage)

	// SendRequest [consumer] sends a request to a connected Thing.
	//
	// Intended for use by consumers when Things are connected using connection reversal.
	//
	// clientID of the device's that hosts the Thing.
	// responseHandler is the optional callback with the response.
	//
	// This returns an error if the Thing is no longer connected.
	SendRequest(clientID string, req *msg.RequestMessage, replyTo msg.ResponseHandler) error

	// SendResponse [Thing] sends the response message over the transport to a remote
	// consumer with the given client and connection ID.
	//
	// Intended for use by Things
	SendResponse(clientID string, cid string, resp *msg.ResponseMessage) error
}

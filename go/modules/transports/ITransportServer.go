package transports

import (
	"time"

	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/msg"
	"github.com/hiveot/hivekit/go/wot/td"
)

// Transport server module definitions for use by transport protocols.
// This contains the protocol types, authentication handler, and interfaces for the shared http server and tls client.

// notifications sent by transport servers to server side services
// These are published by TransportServerBase
const (
	// A client connected to the server
	ConnectedEventName = "connected"
	// A client connection was removed
	DisconnectedEventName = "disconnected"
)

const (
	HiveotSseScProtocolType = "hiveot-ssesc"
	HiveotSseScSubprotocol  = "sse-sc"
	HiveotSseScUriScheme    = "sse"

	HiveotUdsProtocolType = "hiveot-uds"
	HiveotUdsUrlScheme    = "unix"

	HiveotWebsocketProtocolType = "hiveot-websocket"
	HiveotWebsocketSubprotocol  = "hiveot:websocket"
	HiveotWebsocketUriScheme    = "wss"

	WotHttpBasicProtocolType = "http-basic"
	WotHttpBasicSubprotocol  = ""
	WotHttpBasicUriScheme    = "https"

	WotHttpLongPollProtocolType = "http-longpoll"
	WotHttpLongPollSubprotocol  = "longpoll"
	WotHttpLongPollUriScheme    = "https"

	WotMqttProtocolType = "wot-mqtt"
	WotMqttUriScheme    = "mqtts"

	WotSseProtocolType = "wot-sse"
	WotSseUriScheme    = "sse"

	WotWebsocketProtocolType = "wot-websocket"
	WotWebsocketSubprotocol  = "websocket"
	WotWebsocketUriScheme    = "wss"
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

	// GetProtocolType returns type identifier of the server protocol as defined by its module
	GetProtocolType() string

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

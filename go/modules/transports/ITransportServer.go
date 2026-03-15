package transports

import (
	"time"

	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/msg"
	"github.com/hiveot/hivekit/go/wot/td"
)

// Transport server module definitions for use by transport protocols.
// This contains the protocol types, authentication handler, and interfaces for the shared http server and tls client.

// Supported transport protocol types
const (
	// WoT http basic protocol without return channel
	ProtocolSchemeHTTPBasic = "https"
	ProtocolTypeHTTPBasic   = "https"

	// WoT websocket sub-protocol
	ProtocolSchemeWotWSS = "wss"
	ProtocolTypeWotWSS   = "wss"

	// WoT MQTT protocol over WSS
	ProtocolSchemeWotMQTTWSS = "mqtts"
	ProtocolTypeWotMQTTWSS   = "mqtts"

	// HiveOT http SSE subprotocol return channel with direct messaging
	ProtocolSchemeHiveotSSE = "sse"
	ProtocolTypeHiveotSSE   = "sse-sc"

	// HiveOT message envelope passthrough
	ProtocolSchemePassthrough = ""
	ProtocolTypePassthrough   = "passthrough"
)

// notifications sent by transport servers to server side services
// These are published by TransportServerBase
const (
	// A client connected to the server
	ConnectedEventName = "connected"
	// A client connection was removed
	DisconnectedEventName = "disconnected"
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

	// AddTDForms updates the given Thing Description with forms for this transport module.
	AddTDForms(tdoc *td.TD, includeAffordances bool)

	// CloseAll closes all client connections. Mainly intended for testing.
	CloseAll()

	// Return the established connection of the given client, if one exists
	// This returns nil if the client does not have an authenticated connection.
	GetConnectionByClientID(clientID string) IConnection

	// GetConnectURL returns connection URL of the server
	GetConnectURL() string

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

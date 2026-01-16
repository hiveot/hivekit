package transports

import (
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/msg"
	"github.com/hiveot/hivekit/go/wot/td"
)

// Supported transport protocol bindings types
const (
	// WoT http basic protocol without return channel
	ProtocolTypeHTTPBasic = "http-basic"

	// WoT websocket sub-protocol
	ProtocolTypeWotWSS = "wss"

	// WoT MQTT protocol over WSS
	ProtocolTypeWotMQTTWSS = "mqtt-wss"

	// HiveOT http SSE subprotocol return channel with direct messaging
	ProtocolTypeHiveotSSE = "hiveot-sse"

	// HiveOT message envelope passthrough
	ProtocolTypePassthrough = "passthrough"
)

// AuthenticationHandler is the handler for use by transports to authenticate an incoming connection
// and identify the remote client.
//
// If the token is invalid an error is returned
type AuthenticationHandler func(token string) (clientID string, sessionID string, err error)

// A transport module is a server module with hooks for sending messages to remote clients.
type ITransportModule interface {
	modules.IHiveModule

	// AddTDForms updates the given Thing Description with forms for this transport module.
	AddTDForms(tdoc *td.TD, includeAffordances bool)

	// CloseAll closes all client connections. Mainly intended for testing.
	CloseAll()

	// GetConnectURL returns connection URL of the server
	GetConnectURL() string

	// SendNotification [agent] sends a notification over the connection to a consumer.
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

	// Set the handler for authentication connections to this transport module.
	// SetAuthenticationHandler(h AuthenticationHandler)

	// Set the handler for incoming connections
	// SetConnectionHandler(h ConnectionHandler)

}

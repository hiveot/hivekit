package transport

import (
	"time"

	"github.com/hiveot/hivekit/go/api/msg"
)

// Connection status values
type ConnectionStatus string

// connection state machine:
//
//	1: new|lost|closed -> connecting -> connected -> closed
//	2: connecting -> connected -> lost
//	3: connecting -> refused
const (
	// no connection attempt has been made
	StatusNew ConnectionStatus = ""
	// the connection is being established
	StatusConnecting ConnectionStatus = "connecting"
	// the connection was successfully estasblished
	// this is the only status that counts as is-connected.
	StatusConnected ConnectionStatus = "connected"
	// the connection was been closed by the user
	StatusClosed ConnectionStatus = "closed"
	// the connection was dropped or server not reachable
	StatusLost ConnectionStatus = "lost"
	// the connection was refused due to incorrect authentication.
	// reauthentication is required.
	StatusRefused ConnectionStatus = "refused"
)

// ConnectionHandler handles a change in connection status
//
//	status of the connection
//	c is the connection instance being established or disconnected
type ConnectionHandler func(status ConnectionStatus, c IConnection)

// IConnection defines the interfaces of a HiveOT server and client connection.
// Intended for exchanging messages between client and server.
//
// Connections do not differentiate between consumers and devices or services.
// Both clients and servers can provide a connection for use by consumers or agents.
// In case of connection reversal the server can act as the consumer.
//
// All transport servers provide a callback handler that notifies when a new connection
// is received. It is up to the application to handle the connection.
type IConnection interface {

	// Close disconnects the client.
	Close()

	// GetClientID returns the clientID used with authentication
	GetClientID() string

	// Deprecated: this is an artifact slated for deprecation
	// GetConnectionID returns the unique connection ID for this client
	// ConnectionIDs on the server use the clientID to differentiate. Eg clclid.
	GetConnectionID() string

	// Return the client's connecting status
	GetConnectionStatus() ConnectionStatus

	// SendNotification [agent] sends a notification over the connection to a remote consumer.
	// The connection can decide not to deliver the notification depending on subscriptions or
	// other criteria.
	SendNotification(notif *msg.NotificationMessage)

	// SendRequest [consumer] sends a request over the connection to an agent.
	//
	// Since not all connections are bidirectional this interface is unidirectional
	// The system MUST always send an asynchronous response carrying the same correlationID
	// as the request.
	// This returns an error if the request cannot be delivered to the remote side. Once delivered
	// it is the responsibility of the other end to properly forward the request and send a response.
	//
	// Use of IConnection directly by consumers is uncommon. The 'Consumer' helper class provides
	// a SendRequest method that can wait until a response is received. It uses the RnR helper
	// to wait for a response with a matching correlationID.
	SendRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) error

	// SendResponse [agent] sends an asynchronous response over the connection to a consumer.
	// This returns an error if the response could not be delivered.
	SendResponse(response *msg.ResponseMessage) error

	// Change the default timeout for sending messages
	SetTimeout(timeout time.Duration)
}

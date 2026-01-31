package transports

import (
	"time"

	"github.com/hiveot/hivekit/go/msg"
	"github.com/hiveot/hivekit/go/wot/td"
)

const DefaultRpcTimeout = time.Second * 60 // 60 for testing; 3 seconds

// ConnectionHandler handles a change in connection status
//
//	connected is true when connected without errors
//	c is the connection instance being established or disconnected
//	err details why connection failed
type ConnectionHandler func(connected bool, c IConnection, err error)

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

	// ConnectWithToken connects to the transport server using a clientID and
	// corresponding authentication token.
	// This method only applies to client connections. Server side connections will return an error.
	//
	// While most hiveot transport servers support token authentication, the method
	// of obtaining a token depends on the environment. The authn module is intended for this.
	//
	// If a connection is already established on this client then it will be closed first.
	//
	// This connection method must be supported by all client implementations.
	//
	//	clientID is the ID to authenticate as, it must match the token
	//	token is the authentication token obtained on login
	//	ch is the connection handler that is notified when connection is established and disconnects. nil to ignore
	ConnectWithToken(clientID, token string, ch ConnectionHandler) (err error)

	// GetClientID returns the clientID used with authentication
	GetClientID() string

	// GetConnectionID returns the unique connection ID for this client
	// ConnectionIDs on the server use the clientID to differentiate. Eg clclid.
	GetConnectionID() string

	// IsConnected returns the current connection status
	IsConnected() bool

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

	// SetConnectHandler sets the callback for connection status changes
	// This replaces any previously set handler.
	// SetConnectHandler(handler ConnectionHandler)
}

// GetFormHandler is the handler that provides the client with the form needed to invoke an operation
// This returns nil if no form is found for the operation.
type GetFormHandler func(op string, thingID string, name string) *td.Form

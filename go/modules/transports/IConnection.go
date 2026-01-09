package transports

import (
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/msg"
	"github.com/hiveot/hivekit/go/wot/td"
)

// ConnectionInfo provides details of a connection
// type ConnectionInfo struct {

// 	// Connection CA
// 	CaCert *x509.Certificate

// 	// GetClientID returns the authenticated clientID of this connection
// 	ClientID string

// 	// GetConnectionID returns the client's connection ID belonging to this endpoint
// 	ConnectionID string

// 	// GetConnectURL returns the full server URL used to establish this connection
// 	ConnectURL string

// 	// GetProtocolType returns the name of the protocol of this connection
// 	// See ProtocolType... constants above for valid values.
// 	//ProtocolType string

// 	// Connection timeout settings (clients only)
// 	Timeout time.Duration
// }

// ConnectionHandler handles a change in connection status
//
//	connected is true when connected without errors
//	err details why connection failed
//	c is the connection instance being established or disconnected
type ConnectionHandler func(connected bool, err error, c IConnection)

// IConnection defines the interfaces of a server and client connection.
// Intended for exchanging messages between client and server.
//
// Connections do not differentiate between consumers and devices or services.
// Both clients and servers can provide a connection for use by consumers or agents.
// In case of connection reversal the server can act as the consumer.
//
// All transport servers provide a callback handler that notifies when a new connection
// is received. It is up to the application to handle the connection.
// The connections manager module can be used to manage active connections, aggregate
// incoming messages from multiple connections and send messages to connections.
type IConnection interface {

	// Close disconnects the client.
	Close()

	// GetClientID returns the clientID used with authentication
	GetClientID() string

	// GetConnectionID returns the unique connection ID for this client
	// ConnectionIDs on the server use the clientID to differentiate. Eg clclid.
	GetConnectionID() string

	// IsConnected returns the current connection status
	IsConnected() bool

	// SendNotification [agent] sends a notification over the connection to a consumer.
	// The connection can decide not to deliver the notification depending on subscriptions or
	// other criteria.
	// This returns an error if sending the notification was attempted but failed.
	// This returns nil if the notification was delivered or ignored.
	SendNotification(notif *msg.NotificationMessage) error

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
	SetConnectHandler(handler ConnectionHandler)

	// SetNotificationHandler sets the callback for handling received notifications.
	// This replaces any previously set handler.
	//
	// Intended for consumers to receive subscribed notifications.
	// SetNotificationHandler(handler msg.NotificationHandler)

	// SetRequestHandler sets the callback for handling received requests.
	// This replaces any previously set handler.
	//
	// Intended for (device or service) agents to handle requests.
	// SetRequestHandler(handler msg.RequestHandler)

	// SetResponseHandler sets the callback for handling received responses to
	// to asynchronous requests.
	// Intended for consumers to handle responses asynchronously.
	//
	// This replaces any previously set handler.
	// SetResponseHandler(handler msg.ResponseHandler)
}

// GetFormHandler is the handler that provides the client with the form needed to invoke an operation
// This returns nil if no form is found for the operation.
type GetFormHandler func(op string, thingID string, name string) *td.Form

// IClientConnection defines the client interface for establishing connections with a server
// Intended for consumers to connect to a Thing Agent/Hub and for Service agents that connect
// to the Hub.
type IClientConnection interface {
	IConnection

	// ConnectWithClientCert connects to the server using a client certificate.
	// This authentication method is optional
	//ConnectWithClientCert(kp keys.IHiveKey, cert *tls.Certificate) (err error)

	// ConnectWithToken connects to the transport server using a clientID and
	// corresponding authentication token.
	//
	// While most hiveot transport servers support token authentication, the method
	// of obtaining a token depends on the environment.
	//
	// If a connection is already established on this client then it will be closed first.
	//
	// This connection method must be supported by all transport implementations.
	ConnectWithToken(clientID, token string) (err error)

	// Logout from the server
	// This invalidates the authentication token used at login
	// Logout() error

	// Refresh the authentication token and return a new token.
	// This invalidates the authentication token used at login.
	// Refresh() (newToken string, err error)

	// Set the sink for receiving async notifications, requests, and unhandled responses.
	// Intended to be used if the sink is created after the client connection.
	//
	// Async notifications are received when clients subscribe to notifications.
	// Unhandled responses are received when clients do not provide a replyTo to SendRequest.
	// Async requests are received by agents that use reverse connection
	SetSink(sink modules.IHiveModule)
}

// IServerConnection is the interface of an incoming client connection on the server.
// Protocol servers must implement this interface to return information to the consumer.
//
// This provides a return channel for sending messages from the digital twin to
// agents or consumers.
//
// Subscription to events or properties can be made externally via this API,
// or handled internally by the protocol handler if the protocol defines the
// messages for subscription.
type IServerConnection interface {
	IConnection
}

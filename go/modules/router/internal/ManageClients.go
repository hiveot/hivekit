package internal

import (
	"fmt"
	"log/slog"
	"net/url"

	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules/transport"
	"github.com/hiveot/hivekit/go/modules/transport/clients"
)

// GetClientConnection returns a client connection for connecting to the server with
// the given TD. If a connection doesn't exists then create it.
//
// This uses schema://host:port (origin) to identify the connection to use.
// If a connection to this client already exists then use it, otherwise create it.
//
// This returns an error if no connection can be established.
func (m *RouterService) GetClientConnection(tdi *td.TD) (
	c transport.ITransportClient, err error) {

	// use URI scheme to determine the protocol, except for the hiveot WSS, which also
	// has a wss scheme. Instead look at the base path which is fixed.
	protocolType, href := clients.GetProtocolType(tdi)
	parts, err := url.Parse(href)
	if err != nil {
		return nil, err
	}
	// determine the 'origin' for this connection, which is the protocol and address
	// of the connection. Multiple Things from the same agent share the same connection.
	origin := fmt.Sprintf("%s://%s", parts.Scheme, parts.Host)
	m.cmux.Lock()
	c, found := m.deviceConnections[origin]
	if !found {
		// TODO: how to determine the CA for this server?
		// TODO: support use of client cert for this server?
		c, err = clients.NewTransportClient(protocolType, href, m.caCert)
		c.SetTimeout(m.timeout)
		// forward notifications to this module and up to its consumer
		c.SetNotificationSink(m.HandleNotification)
		c.SetConnectHandler(m.HandleClientStatus)
		m.deviceConnections[origin] = c
	}
	m.cmux.Unlock()
	if c.GetConnectionStatus() != transport.StatusConnected {
		err = c.AuthenticateWithForm(tdi, m.credStore.GetCredentials)
		if err == nil {
			err = c.Connect()
		}
		if err != nil {
			err = fmt.Errorf("GetClientConnection. Connection to '%s' failed: %w", origin, err)
			slog.Warn(err.Error())
		}
	}
	return c, err
}

// handle update to connection status
func (m *RouterService) HandleClientStatus(
	oldStatus, newStatus transport.ConnectionStatus, c transport.ITransportClient) {
}

// Return the reverse-client connection to an agent, if it exists.
// This returns nil if the clientID does not have an existing connection.
func (m *RouterService) GetRCConnection(clientID string) (c transport.IConnection) {
	if m.tpServers == nil {
		return nil
	}
	for _, tp := range m.tpServers {
		c := tp.GetConnectionByClientID(clientID)
		if c != nil {
			return c
		}
	}
	return nil
}

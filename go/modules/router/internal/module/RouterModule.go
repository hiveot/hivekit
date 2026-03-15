package module

import (
	"fmt"
	"log/slog"

	"github.com/hiveot/hivekit/go/modules"
	routerapi "github.com/hiveot/hivekit/go/modules/router/api"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/msg"
	"github.com/hiveot/hivekit/go/wot/td"
)

type RouterModule struct {
	modules.HiveModuleBase

	// map of clientID to established connection
	clientConnections map[string]transports.IConnection

	// handler that provides a TD for the given thingID
	getTD func(thingID string) *td.TD

	// transport servers
	tpServers []transports.ITransportServer
}

// Return a client connection to the given href.
//
// If a connection to this client already exists then use it, otherwise create it.
// This returns an error if no connection can be established.
func (m *RouterModule) GetClientConnection(clientID string, href string) (c transports.IConnection, err error) {
	return nil, fmt.Errorf("GetClientConnection: Not yet implemented")
}

// Return the reverse-client connection to an agent, if it exists.
// This returns nil if the clientID does not have an existing connection.
func (m *RouterModule) GetRCConnection(clientID string) (c transports.IConnection) {
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

// HandleRequest handles module requests or routes the request to its destination
func (m *RouterModule) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
	var resp *msg.ResponseMessage

	if req.ThingID != m.GetModuleID() {
		return m.RouteRequest(req, replyTo)
	}
	// handle requests for router module itself
	switch req.Operation {

	// case wot.OpReadProperty:
	// 	resp, err = m.ReadProperty(req)
	// case wot.OpReadMultipleProperties:
	// 	resp, err = m.ReadMultipleProperties(req)
	// case wot.OpReadAllProperties:
	// 	resp, err = m.ReadAllProperties(req)
	// directory specific operations could be handled here
	default:
		err := fmt.Errorf("Unhandled request: thingID='%s', op='%s', name='%s", req.ThingID, req.Operation, req.Name)
		slog.Warn(err.Error())
	}
	if resp != nil {
		err = replyTo(resp)
	}
	return err
}

// Determine if the thing is reachable by the router.
//
// This returns true if a client connection is established by the router, or if
// a reverse connection exists by the thing agent.
func (m *RouterModule) IsReachable(thingID string) bool {
	return false
}

// Return the ISO timestamp when the Thing was last seen by the router.
// This returns an empty string if no known record exists.
func (m *RouterModule) LastSeen(thingID string) string {
	return ""
}

// Route the request to its destination:
//
// Lookup the TD of the ThingID and determine its destination:
//
//  1. If no TD exists then simply forward the request to the request sink
//  2. If the TD contains an agentID, injected by the digitwin module, then lookup
//     the agent's connection to the server and forward the request.
//  3. If the TD points to a non-agent device then establish a connection or re-use
//     an existing connection from the pool.
func (m *RouterModule) RouteRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
	var href string
	tdi := m.getTD(req.ThingID)
	if tdi == nil {
		return m.ForwardRequest(req, replyTo)
	}
	agentID := tdi.GetAgentID()
	forms := tdi.GetForms(req.Operation, req.Name)
	// if no form is found or there is no href, try to find the reverse connection agent
	if len(forms) > 0 {
		// TBD right now just use the first form.
		href = forms[0].GetHRef()
	}
	// without href attempt looking up a reverse connection
	if href == "" {
		c := m.GetRCConnection(agentID)
		if c == nil {
			err = fmt.Errorf("Unable to connection with agent '%s'", agentID)
		} else {
			err = c.SendRequest(req, replyTo)
		}
	} else {
		// get or create an existing client connection
		c, err2 := m.GetClientConnection(agentID, href)
		if c == nil {
			err = fmt.Errorf("Unable to establish a connection to client '%s': %w", agentID, err2)
		} else {
			err = c.SendRequest(req, replyTo)
		}
	}
	return err
}

// Start the router module.
// This currently does nothing.
func (m *RouterModule) Start(_ string) (err error) {
	return err
}

// Stop the router module.
// This closes all established client connections.
func (m *RouterModule) Stop() {
	for clientID, c := range m.clientConnections {
		_ = clientID
		c.Close()
	}
	m.clientConnections = nil
}

// NewRouterModule creates a new router module
//
// getTD is the callback to lookup a TD for a thingID
// transports is a list of transport servers that can contain reverse agent connections.
func NewRouterModule(
	getTD func(thingID string) *td.TD,
	tpServers []transports.ITransportServer) *RouterModule {

	m := &RouterModule{
		getTD:             getTD,
		tpServers:         tpServers,
		clientConnections: make(map[string]transports.IConnection),
	}
	m.SetModuleID(routerapi.DefaultRouterServiceID)

	var _ routerapi.IRouterModule = m // interface check

	return m
}

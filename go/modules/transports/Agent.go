package transports

import (
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/hiveot/hivekit/go/msg"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hivekit/go/wot"
)

// Agent is a helper providing a Golang API for IoT device side WoT operations using the
// standard RRN (request-response-notification) messages. The RRN interface is compatible
// with all HiveKit transport and other modules.
//
// This Agent is intended to link to a transport connection and supports features for
// receiving and responding to requests, publishing events and publishing
// property updates.
//
// IoT devices using Agent are connection agnostics. They can be used in a server configuration
// or as a client to a supporting gateway using connection reversal. See the documentation on agent
// configurations.
//
// An Agent is also a consumer as they are able to invoke services.
type Agent struct {
	*Consumer

	// the application's request handler set with SetRequestHandler
	// intended for sub-protocols that can receive requests. (agents)
	appRequestHandlerPtr atomic.Pointer[msg.RequestHandler]
}

// HandleRequest passes a request to the application request handler and returns the response.
// Handler must be set during init.
// If no handler is set then this fails.
func (ag *Agent) HandleRequest(
	req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {

	// handle requests if any
	hPtr := ag.appRequestHandlerPtr.Load()
	if hPtr != nil {
		err = (*hPtr)(req, replyTo)
	} else {
		// tbd: pass to sink
		// ag.forwardRequestToSink(req, replyTo)
	}
	return
}

// PubActionProgress helper for agents to send a 'running' ActionStatus notification
//
// This sends an ActionStatus message with status of running.
func (ag *Agent) PubActionProgress(req msg.RequestMessage, value any) error {
	status := msg.ActionStatus{
		//AgentID:   ag.GetClientID(),
		ActionID:      req.CorrelationID,
		Input:         req.Input,
		Name:          req.Name,
		Output:        value,
		SenderID:      ag.GetClientID(),
		State:         msg.StatusRunning,
		ThingID:       req.ThingID,
		TimeRequested: req.Created,
		TimeUpdated:   utils.FormatNowUTCMilli(),
	}

	resp := msg.NewNotificationMessage(wot.OpInvokeAction, req.ThingID, req.Name, status)
	return ag.GetConnection().SendNotification(resp)
}

// PubEvent helper for agents to send an event to subscribers.
//
// The underlying transport protocol handles the subscription mechanism.
// The agent itself doesn't track subscriptions.
func (ag *Agent) PubEvent(thingID string, name string, value any) error {

	// This is a response to subscription request.
	// for now assume this is a hub connection and the hub wants all events
	notif := msg.NewNotificationMessage(wot.OpSubscribeEvent, thingID, name, value)
	slog.Info("PubEvent",
		"thingID", thingID,
		"name", name,
		"value", notif.ToString(50),
	)

	return ag.GetConnection().SendNotification(notif)
}

// PubProperty publishes a property change notification to observers.
//
// The underlying transport protocol binding handles the subscription mechanism.
func (ag *Agent) PubProperty(thingID string, name string, value any) error {
	// This is a response to an observation request.
	// send the property update as a response to the observe request
	notif := msg.NewNotificationMessage(wot.OpObserveProperty, thingID, name, value)
	slog.Info("PubProperty",
		"thingID", thingID,
		"name", notif.Name,
		"value", notif.ToString(50),
	)
	return ag.GetConnection().SendNotification(notif)
}

// PubProperties publishes a map of property changes to observers
//
// The underlying transport protocol binding handles the subscription mechanism.
func (ag *Agent) PubProperties(thingID string, propMap map[string]any) error {
	notif := msg.NewNotificationMessage(wot.OpObserveMultipleProperties, thingID, "", propMap)

	slog.Info("PubProperties",
		"thingID", thingID,
		"nrProps", len(propMap),
		"value", notif.ToString(50),
	)
	return ag.GetConnection().SendNotification(notif)
}

// SendNotification sends a property or event notification message
func (ag *Agent) SendNotification(notif *msg.NotificationMessage) error {
	return ag.GetConnection().SendNotification(notif)
}

// SendResponse sends a response for a previous request
func (ag *Agent) SendResponse(resp *msg.ResponseMessage) error {
	return ag.GetConnection().SendResponse(resp)
}

// SetRequestHandler set the application handler for incoming requests
func (ag *Agent) SetRequestHandler(cb msg.RequestHandler) {
	if cb == nil {
		ag.appRequestHandlerPtr.Store(nil)
	} else {
		ag.appRequestHandlerPtr.Store(&cb)
	}
}

// UpdateThing helper for agents to publish an update of a TD in the directory
// Note that this depends on the runtime directory service.
//
// FIXME: change to use directory forms
// func (ag *WotAgent) UpdateThing(tdoc *td.TD) error {
// 	slog.Info("UpdateThing", slog.String("id", tdoc.ID))

// 	// TD is sent as JSON
// 	tdJson, _ := jsoniter.MarshalToString(tdoc)
// 	err := ag.Rpc(wot.OpInvokeAction, ThingDirectoryDThingID, ThingDirectoryUpdateThingMethod,
// 		tdJson, nil)
// 	return err
// }

// NewAgent creates a new agent instance for serving requests and sending responses.
// Since agents are also consumers, they can also send requests and receive responses.
//
// Agents can be connected to when running a server or connect to a hub or gateway as client.
//
// This is a wrapper around the ClientConnection that provides WoT response messages
// publishing properties and events to subscribers and publishing a TD.
func NewAgent(moduleID string,
	cc IClientConnection,
	connHandler ConnectionHandler,
	notifHandler msg.NotificationHandler,
	reqHandler msg.RequestHandler,
	respHandler msg.ResponseHandler,
	timeout time.Duration) *Agent {

	if timeout == 0 {
		timeout = DefaultRpcTimeout
	}
	agent := &Agent{}
	agent.Consumer = NewConsumer(moduleID, cc, timeout)

	agent.SetConnectHandler(connHandler)
	agent.SetNotificationHandler(notifHandler)
	agent.SetRequestHandler(reqHandler)
	agent.SetResponseHandler(respHandler)
	cc.SetConnectHandler(agent.onConnect)
	cc.SetSink(agent)

	return agent
}

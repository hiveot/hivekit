package clients

import (
	"log/slog"
	"sync/atomic"

	"github.com/hiveot/hivekit/go/msg"
	"github.com/hiveot/hivekit/go/utils"
)

// Agent is a helper module providing a Golang API for IoT device side WoT operations using the
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
func (ag *Agent) PubActionProgress(req msg.RequestMessage, value any) {
	status := msg.ResponseMessage{
		//AgentID:   ag.GetClientID(),
		Input:     req.Input,
		Name:      req.Name,
		Output:    value,
		AgentID:   ag.GetClientID(),
		State:     msg.StatusRunning,
		ThingID:   req.ThingID,
		Timestamp: utils.FormatNowUTCMilli(),
	}

	resp := msg.NewNotificationMessage(
		ag.GetClientID(), msg.AffordanceTypeAction, req.ThingID, req.Name, status)
	ag.ForwardNotification(resp)
}

// PubEvent helper for agents to send an event to subscribers.
//
// The underlying transport protocol handles the subscription mechanism.
// The agent itself doesn't track subscriptions.
func (ag *Agent) PubEvent(thingID string, name string, value any) {

	// This is a response to subscription request.
	// for now assume this is a hub connection and the hub wants all events
	notif := msg.NewNotificationMessage(
		ag.GetClientID(), msg.AffordanceTypeEvent, thingID, name, value)
	slog.Info("PubEvent",
		"thingID", thingID,
		"name", name,
		"value", notif.ToString(50),
	)

	ag.ForwardNotification(notif)
}

// PubProperty publishes a property change notification to observers.
//
// The underlying transport protocol binding handles the subscription mechanism.
func (ag *Agent) PubProperty(thingID string, name string, value any) {
	// This is a response to an observation request.
	// send the property update as a response to the observe request
	notif := msg.NewNotificationMessage(
		ag.GetClientID(), msg.AffordanceTypeProperty, thingID, name, value)
	slog.Info("PubProperty",
		"thingID", thingID,
		"name", notif.Name,
		"value", notif.ToString(50),
	)
	ag.ForwardNotification(notif)
}

// PubProperties publishes a map of property changes to observers
//
// The underlying transport protocol binding handles the subscription mechanism.
func (ag *Agent) PubProperties(thingID string, propMap map[string]any) {

	slog.Info("PubProperties",
		"thingID", thingID,
		"nrProps", len(propMap),
	)

	for propName, propVal := range propMap {
		notif := msg.NewNotificationMessage(
			ag.GetClientID(), msg.AffordanceTypeProperty, thingID, propName, propVal)

		ag.ForwardNotification(notif)
	}
}

// SendResponse sends a response for a previous request
// func (ag *Agent) SendResponse(resp *msg.ResponseMessage) error {
// 	return ag.GetConnection().SendResponse(resp)
// }

// SetAppRequestHandler set the application handler for incoming requests
// requests that are not handled are forwarded to the sink.
func (ag *Agent) SetAppRequestHandler(cb msg.RequestHandler) {
	if cb == nil {
		ag.appRequestHandlerPtr.Store(nil)
	} else {
		ag.appRequestHandlerPtr.Store(&cb)
	}
}

// NewAgent creates a new agent (producer) instance for serving requests and sending notifications.
//
//	agentID is the moduleID of this agent
//	appReqHandler is the application handler invoked when receiving requests for this agent.
//
// Since agents are also consumers, they can also send requests and receive responses.
//
// consumers should set this as the sink that handles requests and return notifications
func NewAgent(agentID string, appReqHandler msg.RequestHandler) *Agent {

	agent := &Agent{}
	agent.Consumer = NewConsumer(agentID)

	agent.SetAppRequestHandler(appReqHandler)

	return agent
}

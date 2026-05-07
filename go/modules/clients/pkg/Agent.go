package clientspkg

import (
	"fmt"
	"log/slog"
	"maps"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/factory"
	"github.com/hiveot/hivekit/go/utils"
)

const AgentModuleType = "agent"

// Agent is a module providing a Golang API for IoT device WoT operations using the
// standard RRN (request-response-notification) messages. The RRN interface is compatible
// with all HiveKit modules.
//
// This Agent is intended to be the request sink of a transport connection and supports
// features for receiving and responding to requests, publishing events and publishing
// property updates.
//
// Usage:
//  1. Set this agent as the request sink of a transport connection so it can receive requests
//  2. Set this agent notification sink to the transport connection so it can publish notifications
//  3. Set this agent request sink to other modules that handle server side requests.
//
// Therefore if no appRequestHandler is set, then do not set the request sink to
// the connection for use to send requests.
//
// An Agent is also a consumer as they are able to invoke services.
type Agent struct {
	*Consumer

	// Map of the Things managed by the agent
	tstates map[string]*ThingState
}

// Return the state of a thing managed by the agent
// If no entry for thingID yet exists, one is created.
func (ag *Agent) GetState(thingID string) *ThingState {
	ag.mux.RLock()
	state, ok := ag.tstates[thingID]
	ag.mux.RUnlock()
	if !ok {
		ag.mux.Lock()
		state = NewThingState(thingID)
		ag.tstates[thingID] = state
		defer ag.mux.Unlock()
	}
	return state
}

// HandleReadRequests handles reading of actions, events, and properties for a thing
// managed by this agent.
// This returns nil if the request was handled or an error if this is not a valid read request
// or the thingID is unknown.
func (ag *Agent) HandleReadRequests(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
	var found bool
	var output any

	ag.mux.RLock()
	defer ag.mux.RUnlock()
	state, ok := ag.tstates[req.ThingID]
	if !ok {
		// not handled
		err = fmt.Errorf("Unknown thingID '%s' for agent '%s'", req.ThingID, ag.clientID)
		return err
	}

	switch req.Operation {

	case td.HTOpReadAllEvents:
		output = maps.Clone(state.events)

	case td.OpReadAllProperties:
		props := make(map[string]any)
		for k, notif := range state.properties {
			props[k] = notif.ToString(0)
		}
		output = props

	case td.HTOpReadEvent:
		var key string
		err = req.Decode(&key)
		if err != nil {
			break
		}
		output, found = state.events[key]
		if !found {
			err = fmt.Errorf("Unknown event '%s'", key)
		}

	case td.OpReadProperty:
		var key string
		err = req.Decode(&key)
		if err != nil {
			break
		}
		output, found = state.properties[key]
		if !found {
			err = fmt.Errorf("Unknown property '%s'", key)
		}

	case td.OpReadMultipleProperties:
		var keys []string
		err = req.Decode(&keys)
		if err != nil {
			err = fmt.Errorf("Invalid input: %w", err)
			break
		}
		props := make(map[string]any)
		for _, k := range keys {
			v, ok := state.properties[k]
			if ok {
				props[k] = v
			} else {
				// fail or ignore invalid key? -> graceful degradation
			}
		}
		output = props
	case td.OpQueryAction:
		actionResp, ok := state.actionResponse[req.Name]
		if ok {
			err = fmt.Errorf("Unknown action: %s", req.Name)
		}
		output = actionResp
	case td.OpQueryAllActions:
		output = maps.Clone(state.actionResponse)
	default:
		// not handled
		err = fmt.Errorf("Unhandled operation '%s'", req.Operation)
		return err
	}
	resp := req.CreateResponse(output, err)
	err = replyTo(resp)
	return err
}

// PubActionProgress helper for agents to send a 'running' ActionStatus notification
//
// This sends an ResponseMessage message with status of running.
func (ag *Agent) PubActionProgress(req msg.RequestMessage, value any) {
	status := &msg.ResponseMessage{
		//AgentID:   ag.GetClientID(),
		// Input:     req.Input,
		Name:      req.Name,
		Output:    value,
		SenderID:  ag.GetClientID(),
		Status:    msg.StatusRunning,
		ThingID:   req.ThingID,
		Timestamp: utils.FormatNowUTCMilli(),
	}

	resp := msg.NewNotificationMessage(
		ag.GetClientID(), msg.AffordanceTypeAction, req.ThingID, req.Name, status)

	ag.GetState(req.ThingID).SetActionResponse(req.Name, status)

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
	ag.GetState(thingID).SetEvent(name, notif)

	ag.ForwardNotification(notif)
}

// PubProperty publishes a property change notification to observers.
// This updates the latest value for the property in the state store.
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
	ag.GetState(thingID).SetProperty(name, notif)

	ag.ForwardNotification(notif)
}

// PubProperties publishes a map of property changes to observers
// This updates the latest values for the properties.
//
// The underlying transport protocol binding handles the subscription mechanism.
func (ag *Agent) PubProperties(thingID string, propMap map[string]any) {

	slog.Info("PubProperties",
		"thingID", thingID,
		"nrProps", len(propMap),
	)
	tstate := ag.GetState(thingID)

	for propName, propVal := range propMap {

		notif := msg.NewNotificationMessage(
			ag.GetClientID(), msg.AffordanceTypeProperty, thingID, propName, propVal)

		tstate.SetProperty(propName, notif)

		ag.ForwardNotification(notif)
	}
}

// SendResponse sends a response for a previous request
// func (ag *Agent) SendResponse(resp *msg.ResponseMessage) error {
// 	return ag.GetConnection().SendResponse(resp)
// }

// SetAppRequestHandler set the application handler for incoming requests
// requests that are not handled are forwarded to the sink.
// func (ag *Agent) SetAppRequestHandler(cb msg.RequestHandler) {
// 	if cb == nil {
// 		ag.appRequestHandlerPtr.Store(nil)
// 	} else {
// 		ag.appRequestHandlerPtr.Store(&cb)
// 	}
// }

// NewAgent creates a new agent (producer) instance for serving requests and sending notifications.
//
//	agentID is the moduleID of this agent
//	appReqHandler is the application handler invoked when receiving requests for this agent.
//
// Since agents are also consumers, they can also send requests and receive responses.
//
// consumers should set this as the sink that handles requests and return notifications
func NewAgent(agentID string, appReqHandler msg.RequestHandler) *Agent {

	agent := &Agent{
		tstates: make(map[string]*ThingState),
	}
	agent.Consumer = NewConsumer(agentID)

	if appReqHandler != nil {
		agent.SetAppRequestHook(appReqHandler)
	}
	return agent
}

// Factory for creating an agent module using the factory environment
func NewAgentFactory(f factory.IModuleFactory) (modules.IHiveModule, error) {
	appID := f.GetEnvironment().AppID
	c := NewAgent(appID, nil)
	return c, nil
}

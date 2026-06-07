package agent

import (
	"fmt"
	"log/slog"
	"maps"
	"sync"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/directory"
	"github.com/hiveot/hivekit/go/modules/factory"
	"github.com/hiveot/hivekit/go/utils"
)

const AgentModuleType = "agent"

// Agent is a module providing a Golang API for IoT device operations using the
// standard RRN (request-response-notification) messages. The RRN interface is compatible
// with all HiveKit modules.
//
// This Agent is intended to help building a 'Thing' by:
//   - track Thing status with SetState and GetState
//   - methods for publishing property updates, events, action status and TDs
//     automatic update of property, event and action state when using publish methods
//   - handle read requests for property, event and action status
//   - hook for handling requests directed at the Thing
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
	*modules.HiveModuleBase

	// appRequestHook is the application handler of requests addressed to this module.
	//
	// HandleRequest will invoke this callback or forward requests not destined for
	// this module (moduleID != request.ThingID) to requestSink.
	appRequestHook msg.RequestHandler

	mux sync.RWMutex

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
		err = fmt.Errorf("Unknown thingID '%s' for agent '%s'", req.ThingID, ag.GetThingID())
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
		output, found = state.events[req.Name]
		if !found {
			err = fmt.Errorf("Unknown event '%s'", req.Name)
		}

	case td.OpReadProperty:
		notif, found := state.properties[req.Name]
		// not this returns the last known value so no info on when it changed.
		// TODO: internally hiveot should always work with the full notification
		output = notif.Data
		if !found {
			err = fmt.Errorf("Unknown property '%s'", req.Name)
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

// HandleRequest handles request with thingID set to this moduleID.
//
// If a request hook is set then pass the request to the hook. If the hook does not handle the
// request then it MUST forward it using ForwardRequest.
//
// Applications can also embed this agent and override HandleRequest to handle requests themselves.
//
// Modules that override HandleRequest should first handle the request itself and
// only hand it over to this base method when there is nothing for them to do. This method
// simply forwards the request if no request handler hook is set.
func (m *Agent) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {

	// application can set a hook for handling all requests
	m.mux.RLock()
	handler := m.appRequestHook
	m.mux.RUnlock()

	// invoke registered hook
	if handler != nil {
		err = handler(req, replyTo)
		return err
	}
	if req.ThingID == m.GetThingID() {
		err = m.HandleReadRequests(req, replyTo)
	} else {
		err = m.ForwardRequest(req, replyTo)
	}
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
		SenderID:  ag.GetThingID(),
		Status:    msg.StatusRunning,
		ThingID:   req.ThingID,
		Timestamp: utils.FormatNowUTCMilli(),
	}

	resp := msg.NewNotificationMessage(
		ag.GetThingID(), msg.AffordanceTypeAction, req.ThingID, req.Name, status)

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
		ag.GetThingID(), msg.AffordanceTypeEvent, thingID, name, value)
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
		ag.GetThingID(), msg.AffordanceTypeProperty, thingID, name, value)
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
			ag.GetThingID(), msg.AffordanceTypeProperty, thingID, propName, propVal)

		tstate.SetProperty(propName, notif)

		ag.ForwardNotification(notif)
	}
}

// PubTD publish a request downstream to write a TD to a directory or discovery service.
//
// This addresses the request to the DefaultDirectoryThingID. The directory service
// and the discovery service can both handle the request.
//
// If the application utilizes a reverse connection to a gateway. The request will
// be passed to the gateway where it is routed to the default directory. The TD
// can be a TM as the forms and auth info are not applicable.
//
// If the application runs its own server then it should place a discovery server module
// behind this agent so it can publish the TD. The TD should contain the auth and form
// info for connecting to the server.
func (co *Agent) PubTD(tdJson string) error {

	err := co.Rpc(td.OpInvokeAction,
		directory.DefaultDirectoryThingID,
		directory.ActionCreateThing,
		tdJson, nil)

	return err
}

// Set the hook to invoke when requests are received by this module.
//
// The handler is invoked when requests are received with the ThingID set to
// this agent's moduleID.
//
// This hook is intended to implement agent behavior without having to implement
// a separate module.
//
// The hook MUST either call replyTo with the result or return an error.
// Failure to do so results in the request being lost and the caller waiting
// for a response until timeout.
func (m *Agent) SetAppRequestHook(hook msg.RequestHandler) {
	m.mux.Lock()
	defer m.mux.Unlock()
	m.appRequestHook = hook
}

// NewAgent creates a new agent (producer) instance for serving requests and sending notifications.
//
//	agentID is the ThingID of this agent.
//	appReqHandler is the application handler invoked when receiving requests for this agent.
func NewAgent(agentID string, appReqHandler msg.RequestHandler) *Agent {

	agent := &Agent{
		// agents dont use timeouts
		HiveModuleBase: modules.NewHiveModuleBase(agentID, 0),
		tstates:        make(map[string]*ThingState),
	}

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

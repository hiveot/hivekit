package transports

import (
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/hiveot/hivekit/go/msg"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hivekit/go/wot"
)

// WotAgent is a helper providing a Golang API for IoT device side WoT operations using the
// standard RRN (request-response-notification) messages. The RRN interface is compatible
// with all HiveKit transport and other modules.
//
// This WotAgent features receiving and responding to requests, publishing events and publishgin
// property updates.
//
// Consumer subscriptions are handled by the transport server and no concern of the WotAgent.
//
// IoT devices using WotAgent are connection agnostics. They can be used in a server configuration
// or as a client to a supporting gateway using connection reversal. See the documentation on agent
// configurations.
//
// A WotAgent is also a consumer as they are able to invoke services.
type WotAgent struct {
	*WotConsumer

	// the application's request handler set with SetRequestHandler
	// intended for sub-protocols that can receive requests. (agents)
	appRequestHandlerPtr atomic.Pointer[RequestHandler]
}

// OnRequest passes a request to the application request handler and returns the response.
// Handler must be set by agent subclasses during init.
// This logs an error if no agent handler is set.
func (ag *WotAgent) onRequest(
	req *msg.RequestMessage, c IConnection) *msg.ResponseMessage {

	// handle requests if any
	hPtr := ag.appRequestHandlerPtr.Load()
	if hPtr == nil {
		err := fmt.Errorf("Received request but no handler is set")
		resp := req.CreateResponse(nil, err)
		return resp
	}
	resp := (*hPtr)(req, c)
	return resp
}

// PubActionProgress helper for agents to send a 'running' ActionStatus notification
//
// This sends an ActionStatus message with status of running.
func (ag *WotAgent) PubActionProgress(req msg.RequestMessage, value any) error {
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
func (ag *WotAgent) PubEvent(thingID string, name string, value any) error {

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
func (ag *WotAgent) PubProperty(thingID string, name string, value any) error {
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
func (ag *WotAgent) PubProperties(thingID string, propMap map[string]any) error {
	notif := msg.NewNotificationMessage(wot.OpObserveMultipleProperties, thingID, "", propMap)

	slog.Info("PubProperties",
		"thingID", thingID,
		"nrProps", len(propMap),
		"value", notif.ToString(50),
	)
	return ag.GetConnection().SendNotification(notif)
}

// SendNotification sends a property or event notification message
func (ag *WotAgent) SendNotification(notif *msg.NotificationMessage) error {
	return ag.GetConnection().SendNotification(notif)
}

// SendResponse sends a response for a previous request
func (ag *WotAgent) SendResponse(resp *msg.ResponseMessage) error {
	return ag.GetConnection().SendResponse(resp)
}

// SetRequestHandler set the application handler for incoming requests
func (ag *WotAgent) SetRequestHandler(cb RequestHandler) {
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
func NewWotAgent(cc IConnection,
	connHandler ConnectionHandler,
	notifHandler NotificationHandler,
	reqHandler RequestHandler,
	respHandler ResponseHandler,
	timeout time.Duration) *WotAgent {

	if timeout == 0 {
		timeout = DefaultRpcTimeout
	}

	agent := WotAgent{}
	agent.WotConsumer = NewWotConsumer(cc, timeout)
	agent.SetConnectHandler(connHandler)
	agent.SetNotificationHandler(notifHandler)
	agent.SetRequestHandler(reqHandler)
	agent.SetResponseHandler(respHandler)
	//cc.SetNotificationHandler(agent.onNotification)
	//cc.SetResponseHandler(agent.onResponse)
	//cc.SetConnectHandler(agent.onConnect)
	cc.SetRequestHandler(agent.onRequest)
	return &agent
}

package consumer

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/msg"
)

// Interface of the consumer
type IConsumer interface {
	api.IHiveCell

	// InvokeAction invokes an action on a thing and wait for the response
	// If the response type is known then provide it with output, otherwise use interface{}
	InvokeAction(thingID string, name string, input any, output any) error

	// ObserveProperty sends a request to observe one or all properties
	//
	//	thingID is empty for all things
	//	name is empty for all properties of the selected things
	ObserveProperty(thingID string, name string) error

	// Ping the server and wait for a response.
	// Intended to ensure the server is reachable.
	Ping() error

	QueryAction(thingID, name string) (msg.ActionStatus, error)

	QueryAllActions(thingID string) (map[string]msg.ActionStatus, error)

	ReadAllEvents(thingID string) (map[string]msg.ThingValue, error)

	ReadAllProperties(thingID string) (map[string]msg.ThingValue, error)

	ReadEvent(thingID string, name string) (msg.ThingValue, error)

	ReadProperty(thingID string, name string) (msg.ThingValue, error)

	// Set the hook to invoke with received notifications
	// intended for use when not embedding the consumer but handling
	// it notifications in a separate service.
	// Also works nicely with consumer recipes.
	SetNotificationHook(hook msg.NotificationHandler)

	// Subscribe sends a request for event subscription to the server
	Subscribe(thingID string, name string) error

	// Remove a previous property subscription
	UnobserveProperty(thingID string, name string) error

	// Unsubscribe sends a remove event subscription request to the server
	Unsubscribe(thingID string, name string) error

	// Send a request to write a property value.
	WriteProperty(thingID string, name string, input any, wait bool) error
}

package msg

import (
	"slices"

	"github.com/hiveot/hivekit/go/wot"
)

// Definition of message filters for events properties and actions
type MessageFilter struct {
	// The default retain value if no steps match.
	Events     []MessageFilterStep `yaml:"events"`
	Properties []MessageFilterStep `yaml:"properties"`
	Actions    []MessageFilterStep `yaml:"actions"`
}

// Determine if a notification should be retained based on the filter steps.
// The notification is retained if one step retails.
func (m *MessageFilter) RetainNotification(notif *NotificationMessage) bool {
	switch notif.AffordanceType {
	case AffordanceTypeEvent:
		for _, step := range m.Events {
			if step.Match(notif.ThingID, notif.Name) {
				return step.Retain
			}
		}
	case AffordanceTypeProperty:
		for _, step := range m.Properties {
			if step.Match(notif.ThingID, notif.Name) {
				return step.Retain
			}
		}
	case AffordanceTypeAction:
		for _, step := range m.Actions {
			if step.Match(notif.ThingID, notif.Name) {
				return step.Retain
			}
		}
	}
	// no match
	return false
}

// Determine if a request should be retained based on the filter steps.
// All steps must return true to retain the message.
func (m *MessageFilter) RetainRequest(req *RequestMessage) bool {
	switch req.Operation {
	case wot.OpInvokeAction:
		for _, step := range m.Actions {
			if step.Match(req.ThingID, req.Name) {
				return step.Retain
			}
		}
	case wot.OpWriteProperty:
		for _, step := range m.Properties {
			if step.Match(req.ThingID, req.Name) {
				return step.Retain
			}
		}
	}
	return true
}

// Filter whether to retain an action, property update or event
type MessageFilterStep struct {

	// Optional, the rule applies to data from this (digital twin) Thing
	ThingID string `yaml:"thingID,omitempty"`

	// Optional, the rule applies to property, event or action with these names
	Names []string `yaml:"names,omitempty"`

	// Retain or exclude based on this rule
	Retain bool `yaml:"retain" json:"retain"`
}

// Match returns whether the provided filter parameters match this step.
func (step *MessageFilterStep) Match(thingID string, name string) bool {

	// filters on affordance type, operation, thingID and name
	if step.ThingID != "" {
		if thingID != step.ThingID {
			// thingID provided but does not match
			return false
		}
	}
	// thingID matches or is not specified, which is a match by default
	if step.Names != nil {
		if !slices.Contains(step.Names, name) {
			// names provided but none math this name
			return false
		}
	}
	return true
}

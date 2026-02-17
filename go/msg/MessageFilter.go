package msg

import (
	"log/slog"
	"slices"

	"github.com/hiveot/hivekit/go/wot"
)

// Definition of message filters for events properties and actions
type MessageFilter struct {
	Events     MessageFilterChain `yaml:"events"`
	Properties MessageFilterChain `yaml:"properties"`
	Actions    MessageFilterChain `yaml:"actions"`
}

type MessageFilterChain []MessageFilterStep

// Match returns whether the provided filter parameters pass the chain of steps.
// If the filter is empty (no filter steps are defined) this returns true.
func (chain MessageFilterChain) Accept(thingID string, name string) bool {
	var hasMatch = len(chain) == 0
	for _, step := range chain {
		if step.Match(thingID, name) {
			if !step.Accept {
				return false
			}
			hasMatch = true
		}
	}
	return hasMatch
}

// AcceptNotification determines if a notification is accepted based on the filter steps.
//
// This returns true if no steps are defined or if all matching steps pass.
//
// This iterates a list of steps. Each matching step returns a pass or fail.
// For a notification to pass, all matching steps must pass. If one match is rejected
// then the notification is rejected.
func (f *MessageFilter) AcceptNotification(notif *NotificationMessage) bool {
	if notif == nil {
		slog.Error("AcceptNotification: nil notification")
		return false
	} else if f == nil {
		return true // no filter
	}
	switch notif.AffordanceType {
	case AffordanceTypeEvent:
		return len(f.Events) == 0 || f.Events.Accept(notif.ThingID, notif.Name)
	case AffordanceTypeProperty:
		return len(f.Properties) == 0 || f.Properties.Accept(notif.ThingID, notif.Name)
	case AffordanceTypeAction:
		return len(f.Actions) == 0 || f.Actions.Accept(notif.ThingID, notif.Name)
	}
	return false
}

// AcceptRequest determines if a request passes or is rejected based on the filter steps.
//
// The request passes if all matching steps are accepted.
func (f *MessageFilter) AcceptRequest(req *RequestMessage) bool {
	if f == nil {
		return true // no filter
	}
	switch req.Operation {
	case wot.OpInvokeAction:
		return f.Actions.Accept(req.ThingID, req.Name)
	case wot.OpWriteProperty:
		return f.Properties.Accept(req.ThingID, req.Name)
	}
	return false
}

// Filter whether to retain an action, property update or event
type MessageFilterStep struct {

	// Optional, the rule applies to data from this (digital twin) Thing
	ThingID string `yaml:"thingID,omitempty"`

	// Optional, the rule applies to property, event or action with these names
	Names []string `yaml:"names,omitempty"`

	// Accept or reject based on this rule
	Accept bool `yaml:"accept" json:"accept"`
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

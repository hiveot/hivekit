package msg

import (
	"slices"

	"github.com/hiveot/hivekit/go/wot"
)

type MessageFilter struct {
	Steps []MessageFilterStep `yaml:"steps"`
}

// Determine if a notification should be retained based on the filter steps.
// All steps must return true to retain the message.
func (m *MessageFilter) RetainNotification(notif *NotificationMessage) bool {
	for _, step := range m.Steps {
		if !step.RetainNotification(notif) {
			return false
		}
	}
	return true
}

// Determine if a request should be retained based on the filter steps.
// All steps must return true to retain the message.
func (m *MessageFilter) RetainRequest(req *RequestMessage) bool {
	for _, step := range m.Steps {
		if !step.RetainRequest(req) {
			return false
		}
	}
	return true
}

// Filter whether to retain an action, property update or event
type MessageFilterStep struct {
	// AffordanceType, required: See AffordanceTypeEvent | Property | Action
	AffordanceType AffordanceType `yaml:"messageType"`

	// Optional, the rule applies to data from this (digital twin) Thing
	ThingID string `yaml:"thingID,omitempty"`

	// Optional, the rule applies to property, event or action with these names
	Names []string `yaml:"names,omitempty"`

	// Retain or exclude based on this rule
	Retain bool `yaml:"retain" json:"retain"`
}

// RetainNotification returns true to retain the message, false to exclude it.
func (m *MessageFilterStep) RetainNotification(notif *NotificationMessage) bool {
	// filters on affordance type, operation, thingID and name
	if m.AffordanceType != "" && notif.AffordanceType != m.AffordanceType {
		return false
	}
	if m.ThingID != "" && notif.ThingID != m.ThingID {
		return false
	}
	if m.Names != nil && slices.Contains(m.Names, notif.Name) {
		return false
	}
	return m.Retain
}

// RetainRequest returns true to retain the message, false to exclude it.
func (m *MessageFilterStep) RetainRequest(req *RequestMessage) bool {
	// exclude non-action config or requests
	if m.AffordanceType != AffordanceTypeAction || req.Operation != wot.OpInvokeAction {
		return false
	}

	if m.ThingID != "" && req.ThingID != m.ThingID {
		return false
	}
	if m.Names != nil && slices.Contains(m.Names, req.Name) {
		return false
	}
	return m.Retain
}

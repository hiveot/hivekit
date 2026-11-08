package consumer

import (
	"log/slog"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/wot"
)

// InteractionInput contains the last value and data schema of an input
// Use NewInteractionInput to initialize
type InteractionInput struct {
	// The property, event or action name
	Name string
	// Title with the human name provided by the interaction affordance
	Title string
	// Schema describing the data from property, event or action affordance
	Schema *wot.DataSchema
	// decoded data in its native format as described by the schema
	// eg string, int, array, object
	Value *msg.ThingValue
}

// NewInteractionInput creates a new interaction input for property or action
//
//	tdi Thing Description instance with schemas for the value. Use nil if schema is unknown.
//	name of the input property or action
func NewInteractionInput(tdoc *wot.TD, name string, defaultValue any) *InteractionInput {
	if tdoc == nil {
		return nil
	}

	ii := &InteractionInput{
		Name:  name,
		Value: msg.NewThingValue(defaultValue, "", ""),
	}
	actionAff, found := tdoc.Actions[name]
	if found {
		if actionAff.Input != nil {
			ii.Schema = actionAff.Input
		}
		ii.Title = actionAff.Title
		ii.Value.Unit = ii.Schema.Unit
		return ii
	}
	propAff, found := tdoc.Properties[name]
	if found {
		ii.Schema = &propAff.DataSchema
		ii.Title = propAff.Title
		ii.Value.Unit = ii.Schema.Unit
		return ii
	}

	slog.Warn("NewInteractionInput: name not a property or action input in TD",
		"thingID", tdoc.ID, "name", name)
	return ii
}

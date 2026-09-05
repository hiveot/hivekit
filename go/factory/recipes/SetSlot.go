package recipes

import (
	"fmt"

	"github.com/hiveot/hivekit/go/api"
)

// Replace the slot with the given cell definition.
//
//	recipe to update.
//	name of the slot.
//	cellDef is the definition to place in the slot.
//
// This returns an error if not found
func SetSlot(recipe []api.CellDefinition, name string, slot api.CellDefinition) error {
	found := false
	for i, def := range recipe {
		if def.Type == name {
			found = true
			recipe[i] = slot
			return nil
		}
	}
	if !found {
		return fmt.Errorf("SetSlot. Slot '%s' not found in recipe", name)
	}
	return nil
}

package api

// recipe cell types
const (
	BusRecipeType   = "busRecipe"
	ChainRecipeType = "chainRecipe"
	StarRecipeType  = "starRecipe"
)

// Interface of a cell recipe.
// Recipe constructors are available for a chain and a star formation.
//
// The recipes directory contains templates for various application use-cases such as
// an IoT device running its own server with discover and a IoT device using reverse connections.
// These templates can be used as-is or be copied and modified as seen fit.
type IRecipe interface {
	IHiveCell

	// Place the given cell definition into the recipe slot
	// Originally intended for placing the application cell in the right spot in the chain.
	//
	// This returns an error if the recipe does not contain a slot with the given ID.
	// SetSlot(slotID string, modDef CellDefinition) error

	// Start all the cells in the recipe.
	// Factory recipes instantiate and link cells before calling Start,
	// then start the cells in reverse order so they can send requests.
	// Start() error

	// Stop the factory used by this recipe in reverse order from Start.
	// Stop()
}

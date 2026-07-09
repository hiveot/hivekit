package api

// recipe module types
const (
	BusRecipeType   = "busRecipe"
	ChainRecipeType = "chainRecipe"
	StarRecipeType  = "starRecipe"
)

// Interface of a module recipe.
// Recipe constructors are available for a chain and a star formation.
//
// The recipes directory contains templates for various application use-cases such as
// an IoT device running its own server with discover and a IoT device using reverse connections.
// These templates can be used as-is or be copied and modified as seen fit.
type IRecipe interface {
	IHiveModule

	// Place the given module definition into the recipe slot
	// Originally intended for placing the application module in the right spot in the chain.
	//
	// This returns an error if the recipe does not contain a slot with the given ID.
	SetSlot(slotID string, modDef ModuleDefinition) error

	// Start all the modules in the recipe.
	Start() error

	// Stop the factory used by this recipe
	Stop()
}

package factoryrecipe

import (
	"log/slog"

	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/factory"
)

// Definition of a factory recipe
type FactoryRecipe struct {
	// Map of modules available to the recipe
	ModuleDefs map[string]factory.ModuleDefinition `yaml:"defs"`
	// Chain of modules in the order they are linked
	ModuleChain []string `yaml:"chain"`
	// The factory to use
	f factory.IModuleFactory
}

// Define a module and append it to the chain
func (r *FactoryRecipe) AddModule(moduleType string, moduleDef factory.ModuleDefinition) {
	r.ModuleDefs[moduleType] = moduleDef
	r.ModuleChain = append(r.ModuleChain, moduleType)
}

// Start the modules in this recipe using the given factory
func (r *FactoryRecipe) Start(f factory.IModuleFactory) error {
	r.f = f

	// add the module definitions to the factory
	if r.ModuleDefs != nil {
		// register all modules
		for k, v := range r.ModuleDefs {
			f.RegisterModule(k, v)
		}
	}
	// start and link modules in the defined order
	modList := make([]modules.IHiveModule, 0, len(r.ModuleChain))
	var prevModule modules.IHiveModule
	for _, modType := range r.ModuleChain {
		// getmodule starts it if needed
		m, err := r.f.GetModule(modType)
		modList = append(modList, m)
		if err == nil {
			// Link the module to the previous module in the list
			if prevModule != nil {
				prevModule.SetRequestSink(m.HandleRequest)
				m.SetNotificationSink(prevModule.HandleNotification)
			}
		}
		// oops
		if err != nil {
			slog.Error("StartRecipe: starting module failed. Shutting down", "moduleType", modType)
			f.StopAll()
			return err
		}
		prevModule = m
	}
	return nil
}

// Create a new factory recipe
func NewFactoryRecipe(defs map[string]factory.ModuleDefinition, chain []string) *FactoryRecipe {
	r := &FactoryRecipe{
		ModuleDefs:  defs,
		ModuleChain: chain,
	}
	return r
}

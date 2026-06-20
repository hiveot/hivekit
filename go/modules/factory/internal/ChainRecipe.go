package internal

import (
	"fmt"
	"log/slog"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/factory"
)

// The ChainRecipe is a module that links its modules in a chain formation.
//
// Incoming requests are forwarded to the first module in the chain.
// The last module in the chain will have its request sink set to the sink of the
// chain itself, effectively forwarding requests to the sink of the chain.
//
// On start, all modules are loaded, started and linked in sequence.
//
// The ChainRecipe module itself is registered as the notification sink of the first module
// in the chain and will forward these notifications to its registered notification sink.
type ChainRecipe struct {
	*modules.HiveModuleBase
	// Chain of modules in the order to instantiate and link
	chain []factory.ModuleDefinition `yaml:"chain"`

	// The factory to use
	f factory.IModuleFactory

	// loaded modules in order of the chain
	modList []modules.IHiveModule
}

// Requests sent to the chain are passed on to the first module in the chain.
// If no modules match it is forwarded to the registered sink.
func (m *ChainRecipe) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) error {
	if len(m.modList) == 0 {
		return m.HiveModuleBase.HandleRequest(req, replyTo)
	}
	head := m.modList[0]
	return head.HandleRequest(req, replyTo)
}

// SetSlot sets the given module definition in the chain at the position of the slot.
// Use this before starting the chain.
// Intended to create chain templates where the application module needs to be placed
// before some other modules.
func (m *ChainRecipe) SetSlot(slotID string, modDef factory.ModuleDefinition) error {
	for i, md := range m.chain {
		if md.Type == slotID {
			m.chain[i] = modDef
			return nil
		}
	}
	return fmt.Errorf("SetSlot: slot '%s' not found", slotID)
}

// Start the recipe
func (m *ChainRecipe) Start() error {

	// register all modules with the factory
	for _, moduleDef := range m.chain {
		m.f.RegisterModule(moduleDef)
	}

	// start and link modules in the defined order
	m.modList = make([]modules.IHiveModule, 0, len(m.chain))
	var prevModule modules.IHiveModule
	for _, moduleDef := range m.chain {
		chainedMod, err := m.f.StartModule(moduleDef.Type, true)
		if err != nil {
			slog.Error("StartRecipe: starting module failed. Shutting down",
				"moduleType", moduleDef.Type, "err", err.Error())
			m.Stop()
			return err
		} else if chainedMod == nil {
			// don't track 'one-shot' modules that are used to initialize the factory.
			// These return nil without error.
		} else {
			m.modList = append(m.modList, chainedMod)
			// Link the module to the previous module in the list
			if prevModule != nil {
				prevModule.SetRequestSink(chainedMod)
				chainedMod.SetNotificationSink(prevModule)
			} else {
				// this is the first module, the 'chainRecipe' becomes the notification sink
				chainedMod.SetNotificationSink(m)
			}
		}
		prevModule = chainedMod
	}
	return nil
}

// Create a recipe instance for running modules in a chain formation.
//
// Use Start to instantiate and link the modules in the given order. This uses the factory
// to create the module instances.
// Providing a factory function in the chain is only needed if the factory doesn't contain
// it already.
//
// f is the module factory that instantiates the modules
// chain is a collection of modules in order of instantiation.
//
// This returns the chain recipe module.
func NewChainRecipe(f factory.IModuleFactory, chain []factory.ModuleDefinition) factory.IRecipe {

	m := &ChainRecipe{
		HiveModuleBase: modules.NewHiveModuleBase("", 0),
		f:              f,
		chain:          chain,
	}
	return m
}

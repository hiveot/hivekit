package factorypkg

import (
	"log/slog"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/factory"
)

// The ChainRecipe links its modules in a chain formation.
//
// Incoming requests are forwarded to the first module in the chain.
// The last module in the chain will have its request handler set to the same handler
// of the chain module. Unhandled requests will therefore be forwarded to the request
// sink of the ChainRecipe module itself.
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
		chainMod, err := m.f.GetModule(moduleDef.Type, true)
		if err != nil {
			slog.Error("StartRecipe: starting module failed. Shutting down",
				"moduleType", moduleDef.Type, "err", err.Error())
			m.Stop()
			return err
		} else if chainMod == nil {
			// don't track 'one-shot' modules that are used to initialize the factory.
			// These return nil without error.
		} else {
			m.modList = append(m.modList, chainMod)
			// Link the module to the previous module in the list
			if prevModule != nil {
				prevModule.SetRequestSink(chainMod)
				chainMod.SetNotificationSink(prevModule)
			} else {
				// this is the first module, the 'chainRecipe' becomes the notification sink
				chainMod.SetNotificationSink(m)
			}
		}
		prevModule = chainMod
	}
	return nil
}

// Create a recipe instance for running modules in a chain formation.
//
// f is the module factory that instantiates the modules
//
// This returns the chain recipe module.
func NewChainRecipe(f factory.IModuleFactory, chain []factory.ModuleDefinition) *ChainRecipe {

	m := &ChainRecipe{
		HiveModuleBase: modules.NewHiveModuleBase("", 0),
		f:              f,
		chain:          chain,
	}
	return m
}

package internal

import (
	"fmt"
	"log/slog"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/modules"
)

// The ChainRecipe is a module that links its modules in a chain formation.
//
// Incoming requests are forwarded to the first module in the chain.
// The last module in the chain will have its request sink set to the sink of the
// chain itself, effectively forwarding requests to the sink of the chain.
//
// NOTE: The chain recipe MUST be linked before start. On Start the last module
// of the chain is set to the linked request handler.
//
// On start, all modules are loaded, started and linked in sequence.
//
// The ChainRecipe module itself is registered as the notification sink of the first module
// in the chain and will forward these notifications to its registered notification sink.
type ChainRecipe struct {
	*modules.HiveModuleBase
	// Chain of modules in the order to instantiate and link
	chain []api.ModuleDefinition `yaml:"chain"`

	// The factory to use
	f api.IModuleFactory

	// loaded modules in order of the chain
	modList []api.IHiveModule
}

// Recipe receives notifications from the application.
// Send it up the recipe content chain, starting at the last module.
func (m *ChainRecipe) HandleNotification(notif *msg.NotificationMessage) {
	if len(m.modList) == 0 {
		return
	}
	tail := m.modList[len(m.modList)-1]
	tail.HandleNotification(notif)
}

// Requests sent to the chain are passed on to the first module in the chain.
// If no modules are registered then this is an error.
func (m *ChainRecipe) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) error {
	if len(m.modList) == 0 {
		return fmt.Errorf("HandleRequest: recipe has no modules registered")
	}
	head := m.modList[0]
	return head.HandleRequest(req, replyTo)
}

// Set the sink for notifications from the chain
// This sets the sink to the first module in the chain. Call this after start.
func (m *ChainRecipe) SetNotificationSink(sink api.IHiveModule, thingIDs ...string) {
	if len(m.modList) == 0 {
		slog.Error("SetNotificationSink called but the chain is not started")
		return
	}
	head := m.modList[0]
	head.SetNotificationSink(sink, thingIDs...)
}

// Set the sink for requests from the chain
// This sets the sink to the last module in the chain. Call this after start.
func (m *ChainRecipe) SetRequestSink(sink api.IHiveModule) {
	if len(m.modList) == 0 {
		slog.Error("SetRequestSink called but the chain is not started")
		return
	}
	tail := m.modList[len(m.modList)-1]
	tail.SetRequestSink(sink)
}

// SetSlot sets the given module definition in the chain at the position of the slot.
// Use this before starting the chain.
// Intended to create chain templates where the application module needs to be placed
// before some other modules.
func (m *ChainRecipe) SetSlot(slotID string, modDef api.ModuleDefinition) error {
	for i, md := range m.chain {
		if md.Type == slotID {
			m.chain[i] = modDef
			return nil
		}
	}
	return fmt.Errorf("SetSlot: slot '%s' not found", slotID)
}

// Start the recipe.
// This starts the modules in sequence.
//
// NOTE: The chain recipe must be started before linking to it, as setting the recipe request
// sink sets it on the last module in the chain. and setting the notification sink sets it
// on the first module of the chain:
//
// * linking a request handler sets its as the sink of the last module
// * linking a notification handler sets it as the sink of the first module
// * sending a request to the chain passes it to the first module of the chain
// * sending a notification to the chain passes it to the last module, which makes it
//
//	way to the first module and up to the linked notification handler.
func (m *ChainRecipe) Start() error {

	// register all modules with the factory
	for _, moduleDef := range m.chain {
		m.f.RegisterModule(moduleDef)
	}

	// start and link modules in the defined order
	m.modList = make([]api.IHiveModule, 0, len(m.chain))
	var prevModule api.IHiveModule
	for _, moduleDef := range m.chain {
		member, err := m.f.StartModule(moduleDef.Type, true)
		if err != nil {
			slog.Error("StartRecipe: starting module failed. Shutting down",
				"moduleType", moduleDef.Type, "err", err.Error())
			m.Stop()
			return err
		} else if member == nil {
			// don't track 'one-shot' modules that are used to initialize the factory.
			// These return nil without error.
		} else {
			m.modList = append(m.modList, member)
			// Link the module to the previous module in the list
			if prevModule != nil {
				prevModule.SetRequestSink(member)
				member.SetNotificationSink(prevModule)
			}
		}
		prevModule = member
	}

	return nil
}

// Create a recipe instance for running modules in a chain formation.
//
// Use Start to instantiate and link the modules in the given order. This uses the factory
// to create the module instances.
//
// f is the module factory that instantiates the modules
// chain is a collection of modules in order of instantiation.
//
// This returns the chain recipe module.
func NewChainRecipe(f api.IModuleFactory,
	chain []api.ModuleDefinition) api.IRecipe {

	m := &ChainRecipe{
		HiveModuleBase: modules.NewHiveModuleBase("ChainRecipe", 0),
		f:              f,
		chain:          chain,
	}
	return m
}

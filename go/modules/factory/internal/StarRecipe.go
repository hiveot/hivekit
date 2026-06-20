package internal

import (
	"fmt"
	"log/slog"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/factory"
)

// The StarRecipe is a module that links its modules in a star formation.
//
// Incoming requests are forwarded to the module that matches the request thingID.
// There is no need for linking individual request handlers.
// If a request is received for a thingID not in the star, it is forwarded to the
// star module registered sink.
//
// The star module itself is registered as the notification sink of the modules in the
// star and will forward these notifications to its own registered notification sink.
type StarRecipe struct {
	*modules.HiveModuleBase
	// Chain of modules in the order to instantiate and link
	star []factory.ModuleDefinition `yaml:"star"`

	// The factory to use
	f factory.IModuleFactory

	// module rays by their ThingID
	rays map[string]modules.IHiveModule
}

// Requests sent to the star are passed on to the module with the matching thingID.
// If no modules match it is forwarded to the registered sink.
func (m *StarRecipe) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) error {
	ray, found := m.rays[req.ThingID]
	if found {
		return ray.HandleRequest(req, replyTo)
	}
	return m.HiveModuleBase.HandleRequest(req, replyTo)
}

func (m *StarRecipe) SetSlot(slotID string, modDef factory.ModuleDefinition) error {
	for i, md := range m.star {
		if md.Type == slotID {
			m.star[i] = modDef
			return nil
		}
	}
	return fmt.Errorf("SetSlot: slot '%s' not found", slotID)
}

// Start the recipe
func (m *StarRecipe) Start() error {

	// add the module definitions to the factory
	if m.star != nil {
		// register all modules
		for _, modDef := range m.star {
			m.f.RegisterModule(modDef)
		}
	}
	// start modules in the defined order and link their notifications
	for _, moduleDef := range m.star {
		ray, err := m.f.StartModule(moduleDef.Type, true)
		// module cant be started. This is fatal
		if err != nil {
			slog.Error("StartRecipe: starting module failed. Shutting down",
				"moduleType", moduleDef.Type, "err", err.Error())
			m.Stop()
			return err
		} else if m == nil {
			// don't track 'one-shot' modules that are used to initialize the factory.
			// These return nil without error.
		} else {
			m.rays[ray.GetThingID()] = ray
			// requests send by the ray will be forwarded to the star, which
			// passes it to the ray module with the matching thingID. See HandleRequest.
			ray.SetRequestSink(m)
			// all notifications from the rays will be forwarded to the star. See HandleNotification.
			ray.SetNotificationSink(m)
		}
	}
	return nil
}

// Create a recipe instance for running modules in a star formation.
// This returns the star recipe module.
func NewStarRecipe(
	f factory.IModuleFactory, star []factory.ModuleDefinition) factory.IRecipe {

	m := &StarRecipe{
		HiveModuleBase: modules.NewHiveModuleBase("", 0),
		f:              f,
		star:           star,
	}
	return m
}

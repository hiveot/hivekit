package internal

import (
	"fmt"
	"log/slog"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/modules"
)

// The StarRecipe is a module that links its modules in a star formation.
//
// Incoming requests are forwarded to the module that matches the request thingID.
// There is no need for linking individual request handlers.
//
// If a request is received for a thingID not in the star, it is forwarded to the
// star module registered sink.
//
// The star module itself is registered as the notification sink of the modules in the
// star and will forward these notifications to its own registered notification sink.
type StarRecipe struct {
	*modules.HiveModuleBase
	// modules in the order to instantiate and link
	star []api.ModuleDefinition `yaml:"star"`

	// The factory to use
	f api.IModuleFactory

	// module instances by their ThingID
	instances map[string]api.IHiveModule
}

// Receives notifications from downstream and send it to all modules
func (r *StarRecipe) HandleNotification(notif *msg.NotificationMessage) {
	for _, member := range r.instances {
		member.HandleNotification(notif)
	}
}

// Requests sent to the star are passed on to the module with the matching thingID.
// If no modules match it is forwarded to the registered sink.
func (r *StarRecipe) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) error {
	ray, found := r.instances[req.ThingID]
	if found {
		return ray.HandleRequest(req, replyTo)
	}
	return r.HiveModuleBase.HandleRequest(req, replyTo)
}

func (r *StarRecipe) SetSlot(slotID string, modDef api.ModuleDefinition) error {
	for i, md := range r.star {
		if md.Type == slotID {
			r.star[i] = modDef
			return nil
		}
	}
	return fmt.Errorf("SetSlot: slot '%s' not found", slotID)
}

// Start the recipe
func (r *StarRecipe) Start() error {

	// add the module definitions to the factory
	if r.star != nil {
		// register all modules
		for _, modDef := range r.star {
			r.f.RegisterModule(modDef)
		}
	}
	// start modules in the defined order and link their notifications
	for _, moduleDef := range r.star {
		ray, err := r.f.StartModule(moduleDef.Type, true)
		// module cant be started. This is fatal
		if err != nil {
			slog.Error("StartRecipe: starting module failed. Shutting down",
				"moduleType", moduleDef.Type, "err", err.Error())
			r.Stop()
			return err
		} else if r == nil {
			// don't track 'one-shot' modules that are used to initialize the factory.
			// These return nil without error.
		} else {
			r.instances[ray.GetThingID()] = ray
			// requests send by the ray will be forwarded to the star, which
			// passes it to the ray module with the matching thingID. See HandleRequest.
			ray.SetRequestSink(r)
			// all notifications from the rays will be forwarded to the star. See HandleNotification.
			ray.SetNotificationSink(r)
		}
	}
	return nil
}

// Create a recipe instance for running modules in a star formation.
// This returns the star recipe module.
func NewStarRecipe(
	f api.IModuleFactory, star []api.ModuleDefinition) api.IRecipe {

	m := &StarRecipe{
		HiveModuleBase: modules.NewHiveModuleBase("", 0),
		f:              f,
		star:           star,
	}
	return m
}

package thing

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/teris-io/shortid"
)

const ExposedThingsCellType = "exposed-things"

// ExposedThings is a cell representing an Exposed Thing that manages
// additional exposed things.
//
// Intended to help building a device that manages additional Things such as a zwave
// controller or 1-wire gateway.
//
// This is an exposed thing that handles a query for all things just like
// a directory does.
type ExposedThings struct {
	*ExposedThing

	// Map of managed children Things
	children map[string]*ExposedThing
}

// Add a child Thing and set this Thing as the notification sink.
// If the child has a TD it is published.
func (eThing *ExposedThings) AddChild(child *ExposedThing) {
	eThing.mux.Lock()
	defer eThing.mux.Unlock()
	eThing.children[child.GetID()] = child
	child.SetNotificationSink(eThing)
}

// Return the thing or child that is managed by this cell.
//
// thingID is the ID of a nested Thing or "" for the cell's Thing itself.
func (eThing *ExposedThings) GetThing(thingID string) *ExposedThing {
	eThing.mux.RLock()
	defer eThing.mux.Unlock()

	if thingID == "" || thingID == eThing.GetID() {
		return eThing.ExposedThing
	}

	child, found := eThing.children[thingID]
	if found {
		return child
	}
	return nil
}

// HandleRequest handles exposed thing requests for this thingID or one
// of the children.
//
// If the thingID is unknown then forward the request.
func (eThing *ExposedThings) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {

	if req.ThingID == eThing.GetID() {
		// act as a mini directory
		// experimental, Exposed things expose their own and nested TDs
		// used by the gateway to update the directory with RC devices.
		// if req.Operation == td.OpInvokeAction && req.Name == directory.RetrieveAllThingsAction {
		// 	output := eThing.RetrieveAllThings()
		// 	resp := req.CreateResponse(output, err)
		// 	return replyTo(resp)

		// } else {
		// this one's for me
		return eThing.ExposedThing.HandleRequest(req, replyTo)
		// }
	} else {
		child := eThing.GetThing(req.ThingID)
		if child != nil {
			return child.HandleRequest(req, replyTo)
		}
	}
	err = eThing.ForwardRequest(req, replyTo)
	return err
}

// Return the list of Thing TD's in JSON format.
// Intended for use by consumers/gateway that want to know what is available,
// especially from devices that manage nested Things like zwave, 1-wire, etc.
//
// List of TD's in JSON
// func (eThing *ExposedThings) RetrieveAllThings() []string {
// 	tdoc := eTh
// 	if eThing.tdoc == nil && len(eThing.children) == 0 {
// 		return nil
// 	}
// 	tdList := make([]string, 0)
// 	if eThing.tdocJSON != "" {
// 		tdList = append(tdList, eThing.tdocJSON)
// 	}
// 	for _, nested := range eThing.children {
// 		if nested.tdocJSON != "" {
// 			tdList = append(tdList, nested.tdocJSON)
// 		}
// 	}
// 	return tdList
// }

// NewExposedThing creates a ready to use exposed thing (device or service) instance
// for serving requests and sending notifications.
//
// This handles publishing properties and events, tracks property values,
// and handle property read requests.
//
// A request hook allows an application to use the ExposedThing as a Thing
// and simply hook their request handler into it without embedding.
//
//	thingID is the ID of the exposed Thing.
//	appReqHandler is the application handler invoked when receiving requests for this Thing.
func NewExposedThings(thingID string, appReqHandler msg.RequestHandler) *ExposedThings {

	eThings := &ExposedThings{
		// Things dont send requests so no wait
		ExposedThing: NewExposedThing(thingID, appReqHandler),
	}
	return eThings
}

// Factory for creating an exposed Thing using the factory environment.
//
// This uses the Cell Type name as the thingID prefix followed by shortid.
func NewExposedThingsFactory(f api.ICellFactory, def *api.CellDefinition) (api.IHiveCell, error) {
	thingID := def.Type + "-" + shortid.MustGenerate()
	eThings := NewExposedThings(thingID, nil)
	return eThings, nil
}

package internal

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/directory"
	"github.com/hiveot/hivekit/go/modules/transport/addforms"
	"github.com/teris-io/shortid"
)

// AddFormsServiceImpl modifies TD's sent with directory update and create commands with base, security, and form information from the configured transports.
type AddFormsServiceImpl struct {
	modules.HiveModuleBase

	// Optionally specify a service ID of the directory or discovery service this is addressed to
	// Leave empty to just trigger on the action name.
	dirServiceID string

	// flag, include the forms for all affordances
	includeAffordances bool

	// The callback that returns a list of servers available for connecting to the modules
	getServers func() []api.ITransportServer
}

// convert TDs provided with CreateThing and UpdateThing directory actions
func (m *AddFormsServiceImpl) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) error {
	if req.Operation != td.OpInvokeAction {
		return m.ForwardRequest(req, replyTo)
	}
	if req.Name != directory.CreateThingAction && req.Name != directory.UpdateThingAction {
		return m.ForwardRequest(req, replyTo)
	}
	// if a serviceID is provides it must match that of the request
	if m.dirServiceID != "" && m.dirServiceID != req.ThingID {
		return m.ForwardRequest(req, replyTo)
	}
	tdoc, err := td.UnmarshalTD(req.ToString(0))
	if err != nil {
		return m.ForwardRequest(req, replyTo)
	}

	m.AddTDSecForms(tdoc, m.includeAffordances)

	newInput := td.MarshalTD(tdoc)
	// shallow copy of the request before changing the input
	req2 := *req
	req2.Input = newInput
	return m.ForwardRequest(&req2, replyTo)
}

// Update the base-URL, security scheme and forms to the given TD
func (m *AddFormsServiceImpl) AddTDSecForms(tdoc *td.TD, includeAffordances bool) {
	tpServers := m.getServers()
	for _, srv := range tpServers {
		srv.AddTDSecForms(tdoc, includeAffordances)
	}
}

// NewAddFormsServiceImpl creates a new instance of the service
func NewAddFormsServiceImpl(getServers func() []api.ITransportServer) *AddFormsServiceImpl {
	thingID := addforms.AddFormsModuleType + "-" + shortid.MustGenerate()
	m := &AddFormsServiceImpl{
		HiveModuleBase:     *modules.NewHiveModuleBase(thingID, 0),
		includeAffordances: true,
		getServers:         getServers,
	}
	return m
}

// // NewAddFormsServiceImpl creates a new instance of the service
// func NewAddFormsServiceImpl(tpServers []api.ITransportServer) *AddFormsServiceImpl {
// 	thingID := addforms.AddFormsModuleType + "-" + shortid.MustGenerate()
// 	m := &AddFormsServiceImpl{
// 		HiveModuleBase:     *modules.NewHiveModuleBase(thingID, 0),
// 		includeAffordances: true,
// 		tpServers:          tpServers,
// 	}
// 	return m
// }

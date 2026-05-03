package internal

import (
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/directory"
	"github.com/hiveot/hivekit/go/modules/transports"
)

type AddFormsService struct {
	modules.HiveModuleBase

	// Optionally specify a service ID of the directory or discovery service this is addressed to
	// Leave empty to just trigger on the action name.
	dirServiceID string

	// flag, include the forms for all affordances
	includeAffordances bool

	// The servers available for connecting to the modules
	tpServers []transports.ITransportServer
}

func (m *AddFormsService) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) error {
	if req.Operation != td.OpInvokeAction {
		return m.ForwardRequest(req, replyTo)
	}
	if req.Name != directory.ActionCreateThing && req.Name != directory.ActionUpdateThing {
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

	newInput, _ := td.MarshalTD(tdoc)
	// shallow copy of the request
	req2 := *req
	req2.Input = newInput
	return m.ForwardRequest(&req2, replyTo)
}

// Update the base-URL, security scheme and forms to the given TD
func (m *AddFormsService) AddTDSecForms(tdoc *td.TD, includeAffordances bool) {
	for _, srv := range m.tpServers {
		srv.AddTDSecForms(tdoc, includeAffordances)
	}
}

// NewAddFormsService creates a new instance of the service
func NewAddFormsService(tpServers []transports.ITransportServer) *AddFormsService {
	m := &AddFormsService{
		includeAffordances: true,
		tpServers:          tpServers,
	}
	return m
}

package internal

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/cells"
	"github.com/hiveot/hivekit/go/cells/directory"
	"github.com/hiveot/hivekit/go/cells/transport/addforms"
	"github.com/teris-io/shortid"
)

// AddFormsServiceImpl is a small cell that modifies TD's sent with directory update and
// create commands with base, security, and form information from the configured
// transports.
//
// Intended to be used when TD's need to be modified to add the forms for protocols,
// using provided servers. Use cases:
// 1. In a stand-alone server after the exposed thing and before directory client or discovery server
// 2. In a digital twin gateway where requests for all devices must be redirected to the digital twin.
//
// alternative: add a hook to exposed thing used by the publishTD method to add forms.
type AddFormsServiceImpl struct {
	cells.HiveCellBase

	// Optionally specify a service ID of the directory or discovery service this is addressed to
	// Leave empty to just trigger on the action name.
	dirServiceID string

	// flag, include the forms for all affordances
	includeAffordances bool

	// The callback that returns a list of servers available for connecting to the cells
	getServers func() []api.ITransportServer
}

// Update the base-URL, security scheme and forms to the given TD
func (svc *AddFormsServiceImpl) AddTDSecForms(tdoc *td.TD, includeAffordances bool) {
	tpServers := svc.getServers()
	for _, srv := range tpServers {
		srv.AddTDSecForms(tdoc, includeAffordances)
	}
}

// convert TDs provided with CreateThing and UpdateThing directory actions
func (svc *AddFormsServiceImpl) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) error {
	if req.Operation != td.OpInvokeAction {
		return svc.ForwardRequest(req, replyTo)
	}
	if req.Name != directory.CreateThingAction && req.Name != directory.UpdateThingAction {
		return svc.ForwardRequest(req, replyTo)
	}
	// if the service thingID is set, it must match that of the request
	// without it, thingIDs are ignored.
	if svc.dirServiceID != "" && svc.dirServiceID != req.ThingID {
		return svc.ForwardRequest(req, replyTo)
	}
	tdoc, err := td.UnmarshalTD(req.ToString(0))
	if err != nil {
		return svc.ForwardRequest(req, replyTo)
	}

	svc.AddTDSecForms(tdoc, svc.includeAffordances)

	newInput := td.MarshalTD(tdoc)
	// shallow copy of the request before changing the input
	req2 := *req
	req2.Input = newInput
	return svc.ForwardRequest(&req2, replyTo)
}

// NewAddFormsServiceImpl creates a new instance of the service
func NewAddFormsServiceImpl(getServers func() []api.ITransportServer) *AddFormsServiceImpl {

	thingID := addforms.AddFormsCellType + "-" + shortid.MustGenerate()

	m := &AddFormsServiceImpl{
		HiveCellBase:       *cells.NewHiveCellBase(thingID),
		includeAffordances: true,
		getServers:         getServers,
	}
	return m
}

package internal

import (
	wssapi "github.com/hiveot/hivekit/go/modules/transports/wss/api"
	"github.com/hiveot/hivekit/go/wot"
	"github.com/hiveot/hivekit/go/wot/td"
)

// AddTDForms adds base and forms for use of this protocol to the given TD.
//
// Since the contentType is the default application/json it is omitted
//
// 'includeAffordances' adds forms to all affordances to be compliant with the specifications.
// Btw, this is a waste of space in the TD as it required but not needed with some protocols.
func (srv *WssTransport) AddTDForms(tdoc *td.TD, includeAffordances bool) {
	// 1. Add the base if none is set
	tdoc.Base = srv.GetConnectURL()

	// 2. form for all operations
	// the href is empty because it is the same as base for all forms in this protocol
	form := td.NewForm("", "", wssapi.SubprotocolWotWSS)
	form["op"] = []string{
		wot.OpQueryAllActions,
		wot.OpObserveAllProperties, wot.OpUnobserveAllProperties,
		wot.OpReadAllProperties,
		wot.HTOpReadAllEvents, // hiveot supports reading latest events
		wot.OpSubscribeAllEvents, wot.OpUnsubscribeAllEvents,
	}
	//form["contentType"] = "application/json"
	tdoc.Forms = append(tdoc.Forms, form)

	// Add forms to all affordances to be compliant with the specifications.
	// This is a massive waste of space in the TD.
	if includeAffordances {
		srv.AddAffordanceForms(tdoc)
	}
}

// AddAffordanceForms adds forms to affordances for interacting using the websocket protocol binding
func (srv *WssTransport) AddAffordanceForms(tdoc *td.TD) {
	// websocket have no additional href
	href := ""
	for name, aff := range tdoc.Actions {
		_ = name
		form := td.NewForm("", href, wssapi.SubprotocolWotWSS)
		form["op"] = []string{wot.OpInvokeAction, wot.OpQueryAction}
		aff.AddForm(form)
		// cancel action is currently not supported
	}
	for name, aff := range tdoc.Events {
		_ = name
		form := td.NewForm("", href, wssapi.SubprotocolWotWSS)
		form["op"] = []string{wot.HTOpReadEvent, wot.OpSubscribeEvent, wot.OpUnsubscribeEvent}
		aff.AddForm(form)
	}
	for name, aff := range tdoc.Properties {
		_ = name
		form := td.NewForm("", href, wssapi.SubprotocolWotWSS)
		ops := []string{}
		if !aff.WriteOnly {
			ops = append(ops, wot.OpReadProperty, wot.OpObserveProperty, wot.OpUnobserveProperty)
		}
		if !aff.ReadOnly {
			ops = append(ops, wot.OpWriteProperty)
		}

		form["op"] = ops
		aff.AddForm(form)

	}
}

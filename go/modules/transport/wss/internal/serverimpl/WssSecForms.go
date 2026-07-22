package serverimpl

import (
	"github.com/hiveot/hivekit/go/api/td"
)

// AddTDSecForms updates the TD with base URI, security scheme and forms for use of
// this protocol to the given TD.
//
// Since the contentType is the default application/json it is omitted
//
// 'includeAffordances' adds forms to all affordances to be compliant with the specifications.
// Btw, this is a waste of space in the TD as it required but not needed with some protocols.
func (srv *WssServerImpl) AddTDSecForms(tdoc *td.TD, includeAffordances bool) {
	// 1. Add the base connection endpoint
	// TODO: if this Thing supports multiple protocols it might conflict with
	// the base. In that case base cannot be used and all hrefs must be absolute?
	href := srv.GetConnectURL()
	tdoc.Base = href
	subprotocol := srv.subprotocol

	// 2. Set the security scheme used by the authenticator.
	// TODO: risk of duplicates?
	authr := srv.httpServer.GetAuthenticator()
	authr.AddSecurityScheme(tdoc)

	// 3. add top level form for thing level  operations
	// the href is the connection URL because it is the same as base for all forms in this protocol
	form := td.NewForm("", srv.GetConnectURL())
	form.SetSubprotocol(subprotocol)
	form["op"] = []string{
		td.OpInvokeAction, td.OpCancelAction,
		td.OpQueryAction, td.OpQueryAllActions,

		td.OpReadProperty, td.OpReadAllProperties, td.OpReadMultipleProperties,
		td.OpWriteProperty, td.OpWriteMultipleProperties,
		td.OpObserveProperty, td.OpObserveAllProperties, td.OpObserveMultipleProperties,
		td.OpUnobserveProperty, td.OpUnobserveAllProperties, td.OpUnobserveMultipleProperties,

		// hiveot supports reading latest events
		td.HTOpReadEvent, td.HTOpReadAllEvents,
		td.OpSubscribeEvent, td.OpSubscribeAllEvents,
		td.OpUnsubscribeEvent, td.OpUnsubscribeAllEvents,
	}
	//form["contentType"] = "application/json"
	tdoc.Forms = append(tdoc.Forms, form)

	// 4. Add forms to all affordances to be compliant with the specifications.
	// This does uses the same href to prevent conflict with multiple protocols
	if includeAffordances {

		for _, aff := range tdoc.Actions {
			form := aff.AddForm("", href, "", nil)
			form.SetSubprotocol(subprotocol)
			form["op"] = []string{td.OpInvokeAction, td.OpQueryAction}
		}
		for _, aff := range tdoc.Events {
			form := aff.AddForm("", href, "", nil)
			form.SetSubprotocol(subprotocol)
			form["op"] = []string{td.HTOpReadEvent, td.OpSubscribeEvent, td.OpUnsubscribeEvent}
		}
		for _, aff := range tdoc.Properties {
			form := aff.AddForm("", href, "", nil)
			form.SetSubprotocol(subprotocol)
			if !aff.WriteOnly {
				form["op"] = []string{td.OpReadProperty, td.OpObserveProperty, td.OpUnobserveProperty}
			}
			if !aff.ReadOnly {
				form["op"] = []string{td.OpWriteProperty}
			}
			form["op"] = []string{td.HTOpReadEvent, td.OpSubscribeEvent, td.OpUnsubscribeEvent}
		}
	}
}

package internalserver

import (
	"github.com/hiveot/hivekit/go/api/td"
)

// AddTDForms sets the forms for use of http-basic to the given TD.
//
// This:
//  1. Set TD base to the unix socket
//  2. Set the supported security scheme
//  3. Set Thing level forms for general operations such as readallproperties, queryallactions, ...
//     This is a simple write request payload similar to wss
//  4. Set affordance level forms for property, event and actions if includeAffordance is true
//
// Since content-Type is the default 'application/json' it is omitted as per spec.
func (srv *GrpcServer) AddTDSecForms(tdoc *td.TD, includeAffordances bool) {
	// 1. Add the base connection endpoint
	// TODO: if this Thing supports multiple protocols it might conflict with
	// the base. In that case base cannot be used and all hrefs must be absolute?
	href := srv.GetConnectURL()
	tdoc.Base = href

	// 2. Set the security scheme used by the authenticator.
	// TODO: risk of duplicates?
	srv.authenticator.AddSecurityScheme(tdoc)

	// 3. add top level form for thing level  operations
	// the href is the connection URL because it is the same as base for all forms in this protocol
	form := td.NewForm("", href)
	form["op"] = []string{
		td.OpQueryAllActions,
		td.OpObserveAllProperties, td.OpUnobserveAllProperties,
		td.OpReadAllProperties,
		td.HTOpReadAllEvents, // hiveot supports reading latest events
		td.OpSubscribeAllEvents, td.OpUnsubscribeAllEvents,
	}
	//form["contentType"] = "application/json"
	tdoc.Forms = append(tdoc.Forms, form)

	// 4. Add forms to all affordances to be compliant with the specifications.
	// This does uses the same href to prevent conflict with multiple protocols
	if includeAffordances {

		for _, aff := range tdoc.Actions {
			form := aff.AddForm("", href, "", nil)
			form["op"] = []string{td.OpInvokeAction, td.OpQueryAction}
		}
		for _, aff := range tdoc.Events {
			form := aff.AddForm("", href, "", nil)
			form["op"] = []string{td.HTOpReadEvent, td.OpSubscribeEvent, td.OpUnsubscribeEvent}
		}
		for _, aff := range tdoc.Properties {
			form := aff.AddForm("", href, "", nil)
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

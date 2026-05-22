package internal

import (
	"net/http"

	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/modules/transports/ssesc"
)

// AddTDSecForms updates the TD with base URI, security scheme and forms for use of
// this protocol to the given TD.
//
// Since the contentType is the default application/json it is omitted
//
// 'includeAffordances' adds forms to all affordances to be compliant with the specifications.
// Btw, this is a waste of space in the TD as it required but not needed with some protocols.
func (srv *SseScServer) AddTDSecForms(tdoc *td.TD, includeAffordances bool) {
	// 1. Add the base connection endpoint
	// TODO: if this Thing supports multiple protocols it might conflict with
	// the base. In that case base cannot be used and all hrefs must be absolute?
	href := srv.GetConnectURL()
	tdoc.Base = href
	vars := map[string]string{
		td.UriVarThingID: tdoc.ID,
	}
	// protocolType := transports.ProtocolTypeHiveotSsesc
	subprotocol := transports.SubprotocolHiveotSsesc

	// 2. Set the security scheme used by the authenticator.
	// TODO: risk of duplicates?
	authr := srv.httpServer.GetAuthenticator()
	authr.AddSecurityScheme(tdoc)

	// 3. add thing level form for thing level operations
	// since the payload is a request message, one href for all operations (pub request)
	href2 := ssesc.PostSseScRequestPath
	form := td.NewForm("", href2)
	form.SetSubprotocol(subprotocol)
	form["op"] = []string{
		td.OpQueryAllActions,
		td.OpObserveAllProperties, td.OpUnobserveAllProperties,
		td.OpReadAllProperties,
		td.HTOpReadAllEvents, // hiveot supports reading latest events
		td.OpSubscribeAllEvents, td.OpUnsubscribeAllEvents,
	}
	tdoc.Forms = append(tdoc.Forms, form)

	// 4. Add forms to all affordances to be compliant with the specifications.
	// This does uses the same href to prevent conflict with multiple protocols
	if includeAffordances {

		for _, aff := range tdoc.Actions {
			form := aff.AddForm("", href2, http.MethodPost, vars)
			form.SetSubprotocol(subprotocol)
			form["op"] = []string{td.OpInvokeAction, td.OpQueryAction}
		}
		for _, aff := range tdoc.Events {
			// todo subscribe to events by connecting to endpoint
			form := aff.AddForm("", href2, "", nil)
			form.SetSubprotocol(subprotocol)
			form["op"] = []string{td.HTOpReadEvent, td.OpSubscribeEvent, td.OpUnsubscribeEvent}
		}
		for _, aff := range tdoc.Properties {
			// todo subscribe to props by connecting to endpoint
			form := aff.AddForm("", href2, "", nil)
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

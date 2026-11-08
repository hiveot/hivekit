package internal

import (
	"net/http"

	"github.com/hiveot/hivekit/go/api/wot"
	httpbasictransport "github.com/hiveot/hivekit/go/cells/transport/httpbasic"
	"github.com/hiveot/hivekit/go/utils"
)

// list of supported thing level operations
var thingLevelOperations = []string{
	wot.HTOpPing,
	wot.OpQueryAllActions, wot.OpReadAllProperties, wot.HTOpReadAllEvents}

// list of supported affordance operations
var affordanceOperations = []string{
	wot.HTOpReadEvent,
	wot.OpReadProperty, wot.OpReadMultipleProperties,
	wot.OpWriteProperty, wot.OpWriteMultipleProperties,
	wot.OpInvokeAction, wot.OpQueryAction,
}

// AddTDForms sets the forms for use of http-basic to the given TD.
//
// This:
//  1. Set TD base to the https connection address and port
//  2. Set the supported security scheme
//  3. Set Thing level forms for general operations such as readallproperties, queryallactions, ...
//     The href used is "https://host:port/things/{op}/{id}
//     Where {op} and {id} are replaced with the operation and thingID
//  4. Set affordance level forms for property, event and actions if includeAffordance is true
//     The href used is "https://host:port/things/{op}/{id}/{name}"
//     Where {op} and {id} are replaced with the operation, thingID and affordance name
//
// Since content-Type is the default 'application/json' it is omitted as per spec.
func (srv *HttpBasicServerImpl) AddTDSecForms(tdoc *wot.TD) {
	base := srv.GetConnectURL()
	vars := map[string]string{
		wot.UriVarThingID: tdoc.ID,
	}
	// 1. Add the base connection endpoint
	if tdoc.Base == "" {
		tdoc.Base = base
		base = "" // no need for a full base
	}

	// 2. Set the security scheme used by the authenticator.
	authr := srv.httpServer.GetAuthenticator()
	if authr == nil {
		// cant use the http server without authenticator
		panic("HttpBasicServerImpl requires an authenticator from the http server")
	}
	authr.AddSecurityScheme(tdoc)

	// 3. add thing level form for thing level operations
	// http-basic uses a different href for each operation :(
	for _, op := range thingLevelOperations {
		vars[wot.UriVarOperation] = op
		href := base + httpbasictransport.HttpBasicThingOperationPath
		href = utils.Substitute(href, vars)
		form := wot.NewForm(op, href)
		form.SetMethodName(http.MethodGet)
		tdoc.Forms = append(tdoc.Forms, form)
	}

	// 4. add forms for each affordance
	if srv.includeAffordances {
		affHref := base + httpbasictransport.HttpBasicAffordanceOperationPath
		for name, aff := range tdoc.Actions {
			vars[wot.UriVarName] = name
			aff.AddForm(wot.OpInvokeAction, affHref, http.MethodPost, vars)
			aff.AddForm(wot.OpQueryAction, affHref, http.MethodGet, vars)
		}
		for name, aff := range tdoc.Events {
			vars[wot.UriVarName] = name
			aff.AddForm(wot.HTOpReadEvent, affHref, http.MethodGet, vars)
		}
		for name, aff := range tdoc.Properties {
			vars[wot.UriVarName] = name
			aff.AddForm(wot.OpReadProperty, affHref, http.MethodGet, vars)
			aff.AddForm(wot.OpReadMultipleProperties, affHref, http.MethodGet, vars)
			aff.AddForm(wot.OpWriteProperty, affHref, http.MethodPut, vars)
		}
	}
}

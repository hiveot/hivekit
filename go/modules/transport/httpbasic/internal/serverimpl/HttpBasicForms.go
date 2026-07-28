package serverimpl

import (
	"net/http"

	"github.com/hiveot/hivekit/go/api/td"
	httpbasictransport "github.com/hiveot/hivekit/go/modules/transport/httpbasic"
	"github.com/hiveot/hivekit/go/utils"
)

// list of supported thing level operations
var thingLevelOperations = []string{
	td.HTOpPing,
	td.OpQueryAllActions, td.OpReadAllProperties, td.HTOpReadAllEvents}

// list of supported affordance operations
var affordanceOperations = []string{
	td.HTOpReadEvent,
	td.OpReadProperty, td.OpReadMultipleProperties,
	td.OpWriteProperty, td.OpWriteMultipleProperties,
	td.OpInvokeAction, td.OpQueryAction,
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
func (srv *HttpBasicServerImpl) AddTDSecForms(tdoc *td.TD, includeAffordances bool) {

	base := srv.GetConnectURL()
	vars := map[string]string{
		td.UriVarThingID: tdoc.ID,
	}
	// 1. Add the base connection endpoint
	// TODO: if this Thing supports multiple protocols it might conflict with
	// the base. In that case base cannot be used and all hrefs must be absolute?
	tdoc.Base = base

	// 2. Set the security scheme used by the authenticator.
	// TODO: risk of duplicates?
	authr := srv.httpServer.GetAuthenticator()
	if authr == nil {
		// cant use the http server without authenticator
		panic("HttpBasicServerImpl requires an authenticator from the http server")
	}
	authr.AddSecurityScheme(tdoc)

	// 3. add thing level form for thing level operations
	// http-basic uses a different href for each operation :(
	for _, op := range thingLevelOperations {
		vars[td.UriVarOperation] = op
		href := utils.Substitute(httpbasictransport.HttpBasicThingOperationPath, vars)
		form := td.NewForm(op, href)
		form.SetMethodName(http.MethodGet)
		tdoc.Forms = append(tdoc.Forms, form)
	}

	// 4. add forms for each affordance
	if includeAffordances {
		affHref := httpbasictransport.HttpBasicAffordanceOperationPath
		for name, aff := range tdoc.Actions {
			vars[td.UriVarName] = name
			aff.AddForm(td.OpInvokeAction, affHref, http.MethodPost, vars)
			aff.AddForm(td.OpQueryAction, affHref, http.MethodGet, vars)
		}
		for name, aff := range tdoc.Events {
			vars[td.UriVarName] = name
			aff.AddForm(td.HTOpReadEvent, affHref, http.MethodGet, vars)
		}
		for name, aff := range tdoc.Properties {
			vars[td.UriVarName] = name
			aff.AddForm(td.OpReadProperty, affHref, http.MethodGet, vars)
			aff.AddForm(td.OpReadMultipleProperties, affHref, http.MethodGet, vars)
			aff.AddForm(td.OpWriteProperty, affHref, http.MethodPut, vars)
		}
	}
}

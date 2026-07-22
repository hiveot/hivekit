package serverimpl

import (
	"net/http"

	"github.com/hiveot/hivekit/go/api/td"
	httpbasictransport "github.com/hiveot/hivekit/go/modules/transport/httpbasic"
)

// list of supported thing level operations
var thingLevelOperations = []string{
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
	authr.AddSecurityScheme(tdoc)

	// 3. add thing level form for thing level operations
	// http-basic uses a different href for each operation :(
	for _, op := range thingLevelOperations {
		vars[td.UriVarOperation] = op
		href := tdoc.Substitute(httpbasictransport.HttpBasicThingOperationPath, vars)
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

// GetForm returns a form for the given operation
// // Intended for updating TD's with forms to invoke a request
// func (m *HttpBasicTransport) GetForm(operation string, thingID string, name string) *td.Form {
// 	// TODO: use the standard path /operation/thingID/name
// 	return nil
// }

// // createAffordanceForm returns a form for a thing action/event/property affordance operation
// // the href in the form has the format "{base}/{op}/{id}/{name}
// // where {base} is https://
// // Note: in theory these can be replaced with thing level forms using URI variables, except
// // for the issue that WoT doesn't support this.
// //
// // The baseURL is the URL
// func (srv *HttpBasicServer) createAffordanceForm(op string, httpMethod string,
// 	thingID string, name string) td.Form {

// 	href := fmt.Sprintf("%s/%s/%s/%s", httpbasictransport.HttpBaseFormOp, op, thingID, name)
// 	form := td.NewForm(op, href)
// 	if httpMethod != "" && httpMethod != http.MethodGet {
// 		form.SetMethodName(httpMethod)
// 	}
// 	// contentType has a default of application/json
// 	//form["contentType"] = "application/json"
// 	return form
// }

// // createThingLevelForm returns a form for a thing level http operation
// // the href in the form has the format "https://host:port/things/{op}/{id}
// func (srv *HttpBasicServer) createThingLevelForm(op string, httpMethod string, thingID string) td.Form {
// 	// href is relative to base
// 	href := fmt.Sprintf("%s/%s/%s", httpbasictransport.HttpBaseFormOp, op, thingID)
// 	form := td.NewForm(op, href)
// 	form.SetMethodName(httpMethod)
// 	//form["contentType"] = "application/json"
// 	return form
// }

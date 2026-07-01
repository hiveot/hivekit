package internal

import (
	"fmt"
	"net/http"
	"time"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/directory"
)

const ThingIDURIVar = "thingID"

// DirectoryHttpServer is the module that handlesdirectory requests over http.
// This converts the request to RRN messages and sends it downstream to the directory module.
// It is recommended to place this module before the authorization module.
//
// The http server endpoints follow the specification in:
// https://w3c.github.io/wot-discovery/#exploration-directory-api
type DirectoryHttpServer struct {
	// transport.TransportServerBase
	*modules.HiveModuleBase
	httpServer       api.IHttpServer
	directoryThingID string
}

// AddTDSecForms updates the given Thing Description with security and forms for this
// http endpoint.
func (srv *DirectoryHttpServer) AddTDSecForms(tdoc *td.TD, includeAffordances bool) {
	base := srv.GetConnectURL()

	// 1. Add the base connection endpoint
	// TODO: if this Thing supports multiple protocols it might conflict with
	// the base. In that case base cannot be used and all hrefs must be absolute?
	// tdoc.Base = base

	// 2. Set the security scheme used by the authenticator.
	authenticator := srv.httpServer.GetAuthenticator()
	authenticator.AddSecurityScheme(tdoc)

	// 3. no thing level forms as these operations are defined

	// 4. Set the forms for the actions with uri variables for the thingiD argument
	uriVars := map[string]td.DataSchema{
		"id": {
			AtType: "ThingID",
			Title:  "Thing Description ID",
			Type:   "string",
			Format: "iri-reference",
		}}

	// action: createThing
	aff := tdoc.GetAction(directory.CreateThingAction)
	href := fmt.Sprintf("%s/things/{id}", base)
	f := aff.AddForm(td.OpInvokeAction, href, http.MethodPost, nil)
	f["response"] = map[string]any{
		"description":         "Success created new resource",
		"htv:statusCodeValue": 201, // 201 ideally returns new content
	}
	aff.UriVariables = uriVars

	// action: deleteThing
	aff = tdoc.GetAction(directory.DeleteThingAction)
	aff.UriVariables = uriVars
	href = fmt.Sprintf("%s/things/{id}", base)
	f = aff.AddForm(td.OpInvokeAction, href, http.MethodDelete, nil)
	f["response"] = map[string]any{
		"description":         "Success with no content",
		"htv:statusCodeValue": 204,
		"contentType":         "application/td+json",
	}

	// action: retrieveAllThings
	aff = tdoc.GetAction(directory.RetrieveAllThingsAction)
	href = fmt.Sprintf("%s/things", base)
	f = aff.AddForm(td.OpInvokeAction, href, http.MethodGet, nil)
	f["response"] = map[string]any{
		"description":         "Success with response",
		"htv:statusCodeValue": 200,
		"contentType":         "application/td+json",
	}

	// action: retrieveThing
	aff = tdoc.GetAction(directory.RetrieveThingAction)
	aff.UriVariables = uriVars
	href = fmt.Sprintf("%s/things/{id}", base)
	f = aff.AddForm(td.OpInvokeAction, href, http.MethodGet, nil)
	f["response"] = map[string]any{
		"description":         "Success with response",
		"htv:statusCodeValue": 200,
		"contentType":         "application/td+json",
	}

	// action: updateThing
	aff = tdoc.GetAction(directory.UpdateThingAction)
	aff.UriVariables = uriVars
	href = fmt.Sprintf("%s/things/{id}", base)
	f = aff.AddForm(td.OpInvokeAction, href, http.MethodPut, nil)
	f["response"] = map[string]any{
		"description":         "Success new or updated resource; no content",
		"htv:statusCodeValue": 201,
		"contentType":         "application/td+json",
	}

}

// CloseAll force-closes all connections.
// This is a ITransportServer api that does nothing here
func (srv *DirectoryHttpServer) CloseAll() {
}

// Return the base URI this endpoint is listening on
// Intended for inclusion in the directory TDD
func (srv *DirectoryHttpServer) GetConnectURL() string {
	baseURI := srv.httpServer.GetConnectURL()
	return baseURI
}

// ITransportServer stub - not supported in uni-directional transports
func (srv *DirectoryHttpServer) GetConnectionByConnectionID(clientID, connectionID string) (c api.IConnection) {
	return nil
}

// ITransportServer stub - not supported in uni-directional transports
func (srv *DirectoryHttpServer) GetConnectionByClientID(clientID string) (c api.IConnection) {
	return nil
}

// ITransportServer stub - not supported in uni-directional transports
func (srv *DirectoryHttpServer) SendNotification(notif *msg.NotificationMessage) {
}

// ITransportServer stub - not supported in uni-directional transports
func (srv *DirectoryHttpServer) SendRequest(
	senderID string, req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
	return fmt.Errorf("SendRequest: Not supported")
}

// ITransportServer stub - not supported in uni-directional transports
func (srv *DirectoryHttpServer) SendResponse(
	clientID, cid string, resp *msg.ResponseMessage) (err error) {
	return fmt.Errorf("SendResposne: not supported")
}

// Start a new Directory HTTP handler and start listening on the given router
//
// This panics if no http server is provided.
//
// This registers the HTTP API with the router and serves its TD on the
// .well-known/wot endpoint as per discovery specification.
//
//	httpServer to register with
//	respTimeout is the maximum time the server waits for a response when forwarding directory requests
//	 to the directory server.
func StartDirectoryHttpServer(httpServer api.IHttpServer, respTimeout time.Duration) *DirectoryHttpServer {

	if httpServer == nil {
		panic("NewDirectoryHttpServer: Missing http server")
	}

	srv := &DirectoryHttpServer{
		HiveModuleBase:   modules.NewHiveModuleBase("DirectoryHttpServer", respTimeout),
		httpServer:       httpServer,
		directoryThingID: directory.DefaultDirectoryThingID,
	}
	protRoute := httpServer.GetProtectedRoute()
	// add secured routes
	// protRoute.Get(directory.WellKnownWoTPath, srv.handleRetrieveTDD)

	protRoute.Get("/things", srv.handleRetrieveAllThings)
	thingPath := fmt.Sprintf("/things/{%s}", ThingIDURIVar)
	protRoute.Post(thingPath, srv.handleCreateThing)
	protRoute.Get(thingPath, srv.handleRetrieveThing)
	protRoute.Put(thingPath, srv.handleUpdateThing)
	protRoute.Delete(thingPath, srv.handleDeleteThing)

	var _ directory.IDirectoryHttpServer = srv
	return srv
}

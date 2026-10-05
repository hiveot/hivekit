package internal

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/cells"
	"github.com/hiveot/hivekit/go/cells/directory"
	"github.com/hiveot/hivekit/go/utils"
)

// DirectoryHttpServerImpl is the service that handles directory requests over http.
// This converts the request to RRN messages and sends it downstream to the directory service.
// It is recommended to place this before the authorization service.
//
// The http server endpoints follow the specification in:
// https://w3c.github.io/wot-discovery/#exploration-directory-api
type DirectoryHttpServerImpl struct {
	*cells.HiveCellBase
	httpServer       api.IHttpServer
	directoryThingID string
	// the TD describing this server
	serverTD *td.TD
}

// AddTDSecForms updates the given Thing Description with security and forms for
// this http endpoint.
//
//	tdoc the TD to update
//	includeAffordances is ignored
func (srv *DirectoryHttpServerImpl) AddTDSecForms(tdoc *td.TD, includeAffordances bool) {
	base := srv.GetConnectURL()

	// 1. Add the base connection endpoint
	// TODO: if this Thing supports multiple protocols it might conflict with
	// the base. In that case base cannot be used and all hrefs must be absolute?
	// tdoc.Base = base

	// FIXME: this only needs to add forms to the directory TD, not any others.

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
	href = fmt.Sprintf("%s/things?limit={limit}&offset={offset}", base)
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
func (srv *DirectoryHttpServerImpl) CloseAll() {
}

// Return the base URI this endpoint is listening on
// Intended for inclusion in the directory TDD
func (srv *DirectoryHttpServerImpl) GetConnectURL() string {
	baseURI := srv.httpServer.GetConnectURL()
	return baseURI
}

// ITransportServer stub - not supported in uni-directional transports
func (srv *DirectoryHttpServerImpl) GetConnectionByConnectionID(clientID, connectionID string) (c api.IConnection) {
	return nil
}

// ITransportServer stub - not supported in uni-directional transports
func (srv *DirectoryHttpServerImpl) GetConnectionByClientID(clientID string) (c api.IConnection) {
	return nil
}

// GetTD returns the server TD, containing connection and authentication information
func (srv *DirectoryHttpServerImpl) GetTD() *td.TD {
	return srv.serverTD
}

// handleCreateThing creates a new TD in the directory
//
// Only devices, services and admin should be allowed to update the TD. This can be handled by authz.
// The thingID must contain the device ID as the prefix to ensure unique namespace,
// so the stored ThingID will be deviceID:thingID.
func (srv *DirectoryHttpServerImpl) handleCreateThing(w http.ResponseWriter, r *http.Request) {

	rp, err := srv.httpServer.GetRequestParams(r)
	if err == nil {
		tdJson := string(rp.Payload) // ensure correct serialization of payload
		req := msg.NewRequestMessage(
			td.OpInvokeAction, srv.directoryThingID, directory.CreateThingAction, tdJson)
		req.SenderID = rp.ClientID
		_, err = srv.EmitRequestWait(req)
	}
	utils.WriteReply(w, true, nil, err) // 201
}

func (srv *DirectoryHttpServerImpl) handleDeleteThing(w http.ResponseWriter, r *http.Request) {
	// A thingID is provided otherwise this handler would not have been called
	rp, err := srv.httpServer.GetRequestParams(r)
	thingID := chi.URLParam(r, td.UriVarThingID)

	req := msg.NewRequestMessage(
		td.OpInvokeAction, srv.directoryThingID, directory.DeleteThingAction, thingID)
	req.SenderID = rp.ClientID
	_, err = srv.EmitRequestWait(req)

	utils.WriteReply(w, true, nil, err) // 204
}

func (srv *DirectoryHttpServerImpl) handleRetrieveThing(w http.ResponseWriter, r *http.Request) {
	var resp *msg.ResponseMessage
	// A thingID is provided otherwise this handler would not have been called
	thingID := chi.URLParam(r, td.UriVarThingID)
	rp, err := srv.httpServer.GetRequestParams(r)
	if err == nil {
		req := msg.NewRequestMessage(
			td.OpInvokeAction, srv.directoryThingID, directory.RetrieveThingAction, thingID)
		req.SenderID = rp.ClientID
		resp, err = srv.EmitRequestWait(req)
	}
	if err != nil {
		utils.WriteError(w, err, 0)
		return
	}
	var tdocJson string
	err = resp.Decode(&tdocJson)
	w.Write([]byte(tdocJson)) // 200
}

func (srv *DirectoryHttpServerImpl) handleRetrieveAllThings(w http.ResponseWriter, r *http.Request) {
	var resp *msg.ResponseMessage
	rp, err := srv.httpServer.GetRequestParams(r)
	if err == nil {
		qp := r.URL.Query()
		offsetStr := qp.Get("offset")
		limitStr := qp.Get("limit")
		offset, _ := strconv.ParseInt(offsetStr, 10, 32)
		limit, _ := strconv.ParseInt(limitStr, 10, 32)
		args := directory.RetrieveAllThingsArgs{
			Offset: int(offset),
			Limit:  int(limit),
		}
		req := msg.NewRequestMessage(
			td.OpInvokeAction, srv.directoryThingID, directory.RetrieveAllThingsAction, args)
		req.SenderID = rp.ClientID
		resp, err = srv.EmitRequestWait(req)
	}
	utils.WriteReply(w, true, resp.Output, err) // 200
}

// handleUpdateThing handle http request to update a Thing's TD
//
// Only device and admin should be allowed to update the TD. This can be handled by authz.
// The thingID must contain the device accountID as the prefix to ensure unique namespace,
// so the stored ThingID will be deviceID:thingID.
func (srv *DirectoryHttpServerImpl) handleUpdateThing(w http.ResponseWriter, r *http.Request) {
	var resp *msg.ResponseMessage
	rp, err := srv.httpServer.GetRequestParams(r)
	if err == nil {
		tdJson := string(rp.Payload) // ensure correct serialization of payload
		req := msg.NewRequestMessage(
			td.OpInvokeAction, srv.directoryThingID, directory.UpdateThingAction, tdJson)
		req.SenderID = rp.ClientID
		resp, err = srv.EmitRequestWait(req)
		_ = resp
	}
	utils.WriteReply(w, true, nil, err) // 201
}

// ITransportServer stub - not supported in uni-directional transports
func (srv *DirectoryHttpServerImpl) SendNotification(notif *msg.NotificationMessage) {
}

// ITransportServer stub - not supported in uni-directional transports
func (srv *DirectoryHttpServerImpl) SendRequest(
	senderID string, req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
	return fmt.Errorf("SendRequest: Not supported")
}

// ITransportServer stub - not supported in uni-directional transports
func (srv *DirectoryHttpServerImpl) SendResponse(
	clientID, cid string, resp *msg.ResponseMessage) (err error) {
	return fmt.Errorf("SendResposne: not supported")
}

// Return a ready-to-use Directory HTTP handler using the given http server.
//
// This converts directory http requests to RRN messages.
//
//	dirThingID ThingID of the directory service
//	httpServer to serve the directory http requests
//	respTimeout is the maximum time the server waits for a response when forwarding directory requests
//	 to the directory server.
func NewDirectoryHttpServerImpl(
	dirThingID string, httpServer api.IHttpServer, respTimeout time.Duration) (*DirectoryHttpServerImpl, error) {

	if httpServer == nil {
		err := fmt.Errorf("NewDirectoryHttpServer: httpserver is nil")
		return nil, err
	}

	srv := &DirectoryHttpServerImpl{
		HiveCellBase:     cells.NewHiveCellBase("DirectoryHttpServer"),
		httpServer:       httpServer,
		directoryThingID: dirThingID,
	}

	protRoute := httpServer.GetProtectedRoute()
	// add secured routes
	// protRoute.Get(directory.WellKnownWoTPath, srv.handleRetrieveTDD)

	protRoute.Get("/things", srv.handleRetrieveAllThings)
	thingPath := fmt.Sprintf("/things/{%s}", td.UriVarThingID)
	protRoute.Post(thingPath, srv.handleCreateThing)
	protRoute.Get(thingPath, srv.handleRetrieveThing)
	protRoute.Put(thingPath, srv.handleUpdateThing)
	protRoute.Delete(thingPath, srv.handleDeleteThing)

	var _ directory.IDirectoryHttpServer = srv
	return srv, nil
}

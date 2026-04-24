package directoryhttp

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/directory"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/utils"
)

const ThingIDURIVar = "thingID"

// DirectoryHttpHandler is the module that handlesdirectory requests over http.
// This converts the request to RRN messages and sends it downstream to the directory module.
// It is recommended to place this module before the authorization module.
type DirectoryHttpHandler struct {
	modules.HiveModuleBase
	httpServer       transports.IHttpServer
	directoryThingID string
}

// AddTDSecForms updates the given Thing Description with security and forms for this
// http endpoint.
func (srv *DirectoryHttpHandler) AddTDSecForms(tdoc *td.TD, includeAffordances bool) {
	authenticator := srv.httpServer.GetAuthenticator()
	authenticator.AddSecurityScheme(tdoc)
	// FIXME: add forms - maybe use the http-basic server instead of http server?
}

// Return the base URI this endpoint is listening on
// Intended for inclusion in the directory TDD
func (srv *DirectoryHttpHandler) GetBaseURL() string {
	baseURI := srv.httpServer.GetConnectURL()
	return baseURI
}

func (srv *DirectoryHttpHandler) handleDeleteThing(w http.ResponseWriter, r *http.Request) {
	// A thingID is provided otherwise this handler would not have been called
	rp, err := srv.httpServer.GetRequestParams(r)
	thingID := chi.URLParam(r, ThingIDURIVar)

	req := msg.NewRequestMessage(rp.ClientID,
		td.OpInvokeAction, srv.directoryThingID, directory.ActionDeleteThing, thingID, "")
	_, err = srv.ForwardRequestWait(req)

	utils.WriteReply(w, true, nil, err)
}

// Read the directory service TD itself
func (srv *DirectoryHttpHandler) handleReadDirectoryTD(w http.ResponseWriter, r *http.Request) {
	var tddJson string

	rp, err := srv.httpServer.GetRequestParams(r)
	if err == nil {
		err = srv.Rpc(rp.ClientID, td.OpInvokeAction,
			srv.directoryThingID, directory.ActionRetrieveTDD, nil, &tddJson)
	}
	if err != nil {
		utils.WriteError(w, err, 0)
		return
	}
	// set the http server as the base URL.
	// FIXME: where are security and forms added?
	tm := string(tddJson)
	tdi, err := td.UnmarshalTD(tm)
	_ = err
	tdi.Base = srv.httpServer.GetConnectURL()
	utils.WriteReply(w, true, tdi, nil)
}

func (srv *DirectoryHttpHandler) handleRetrieveThing(w http.ResponseWriter, r *http.Request) {
	var tdJson string
	// A thingID is provided otherwise this handler would not have been called
	thingID := chi.URLParam(r, ThingIDURIVar)
	rp, err := srv.httpServer.GetRequestParams(r)
	if err == nil {
		err = srv.Rpc(rp.ClientID, td.OpInvokeAction,
			srv.directoryThingID, directory.ActionRetrieveThing, thingID, &tdJson)
	}
	if err != nil {
		utils.WriteError(w, err, 0)
		return
	}
	w.Write([]byte(tdJson))
}

func (srv *DirectoryHttpHandler) handleRetrieveAllThings(w http.ResponseWriter, r *http.Request) {
	var tdList []string
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
		err = srv.Rpc(rp.ClientID, td.OpInvokeAction,
			srv.directoryThingID, directory.ActionRetrieveAllThings, args, &tdList)
	}
	utils.WriteReply(w, true, tdList, err)
}

// handleUpdateThing handle http request to update a Thing's TD
//
// Only agents and admin should be allowed to update the TD. This can be handled by authz.
// The thingID must contain the agent as the prefix to ensure unique namespace,
// so the stored ThingID will be agentID:thingID.
func (srv *DirectoryHttpHandler) handleUpdateThing(w http.ResponseWriter, r *http.Request) {
	var tdJson string
	rp, err := srv.httpServer.GetRequestParams(r)
	if err == nil {
		tdJson = string(rp.Payload)
		err = srv.Rpc(rp.ClientID, td.OpInvokeAction,
			srv.directoryThingID, directory.ActionUpdateThing, tdJson, nil)
	}
	utils.WriteReply(w, true, nil, err)
}

// Create a new Directory HTTP handler and start listening on the given router
//
// This registers the HTTP API with the router and serves its TD on the
// .well-known/wot endpoint as per discovery specification.
func StartDirectoryHttpHandler(httpServer transports.IHttpServer) *DirectoryHttpHandler {
	srv := &DirectoryHttpHandler{
		httpServer:       httpServer,
		directoryThingID: directory.DefaultDirectoryThingID,
	}
	protRoute := httpServer.GetProtectedRoute()
	// add secured routes
	protRoute.Get(directory.WellKnownWoTPath, srv.handleReadDirectoryTD)

	protRoute.Get("/things", srv.handleRetrieveAllThings)
	thingPath := fmt.Sprintf("/things/{%s}", ThingIDURIVar)
	protRoute.Get(thingPath, srv.handleRetrieveThing)
	protRoute.Put(thingPath, srv.handleUpdateThing)
	protRoute.Delete(thingPath, srv.handleDeleteThing)

	var _ directory.IDirectoryHttpServer = srv
	return srv
}

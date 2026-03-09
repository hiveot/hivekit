// Package directoryserver with the WoT defined REST API
package directoryserver

import (
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	directoryapi "github.com/hiveot/hivekit/go/modules/directory/api"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hivekit/go/wot/td"
)

const ThingIDURIVar = "thingID"

// Handle directory HTTP requests for reading and writing the Thing directory as
// defined in the WoT discovery specification.
// This uses the given chi router which should have authentication/authorization
// middleware installed.
type DirectoryRestHandler struct {
	service    directoryapi.IDirectoryServer
	httpServer transports.IHttpServer
}

// Read the directory service TD itself
func (srv *DirectoryRestHandler) handleReadDirectoryTD(w http.ResponseWriter, r *http.Request) {
	tm := string(DirectoryTMJson)
	tdi, err := td.UnmarshalTD(tm)
	_ = err
	// FIXME: this is not the best way to convert the TM to TD
	tdi.Base = srv.httpServer.GetConnectURL()
	utils.WriteReply(w, true, tdi, nil)
}

func (srv *DirectoryRestHandler) handleRetrieveThing(w http.ResponseWriter, r *http.Request) {
	// A thingID is provided otherwise this handler would not have been called
	thingID := chi.URLParam(r, ThingIDURIVar)
	tdJSON, err := srv.service.RetrieveThing(thingID)
	if err != nil {
		utils.WriteError(w, err, 0)
	} else {
		w.Write([]byte(tdJSON))
	}
}

func (srv *DirectoryRestHandler) handleDeleteThing(w http.ResponseWriter, r *http.Request) {
	// A thingID is provided otherwise this handler would not have been called
	thingID := chi.URLParam(r, ThingIDURIVar)

	rp, err := srv.httpServer.GetRequestParams(r)
	if err == nil {
		err = srv.service.DeleteThing(rp.ClientID, thingID)
	}
	utils.WriteReply(w, true, nil, err)
}

func (srv *DirectoryRestHandler) handleRetrieveAllThings(w http.ResponseWriter, r *http.Request) {
	qp := r.URL.Query()
	offsetStr := qp.Get("offset")
	limitStr := qp.Get("limit")
	offset, _ := strconv.ParseInt(offsetStr, 10, 32)
	limit, _ := strconv.ParseInt(limitStr, 10, 32)

	tdJSON, err := srv.service.RetrieveAllThings(int(offset), int(limit))
	utils.WriteReply(w, true, tdJSON, err)
}

// handleUpdateThing handle http request to update a Thing's TD
//
// Only agents and admin are allowed to update the TD.
// The thingID must contain the agent as the prefix to ensure unique namespace,
// so the stored ThingID will be agentID:thingID.
func (srv *DirectoryRestHandler) handleUpdateThing(w http.ResponseWriter, r *http.Request) {

	tdJsonBin, err := io.ReadAll(r.Body)
	tdJson := string(tdJsonBin)

	rp, err := srv.httpServer.GetRequestParams(r)
	if err == nil {
		err = srv.service.UpdateThing(rp.ClientID, tdJson)
	}
	utils.WriteReply(w, true, nil, err)
}

// Create a new Directory REST handler and start listening on the given router
func StartDirectoryRestHandler(service directoryapi.IDirectoryServer, httpServer transports.IHttpServer) *DirectoryRestHandler {
	srv := &DirectoryRestHandler{
		httpServer: httpServer,
		service:    service,
	}
	protRoute := httpServer.GetProtectedRoute()
	// add secured routes
	protRoute.Get(directoryapi.WellKnownWoTPath, srv.handleReadDirectoryTD)

	protRoute.Get("/things", srv.handleRetrieveAllThings)
	thingPath := fmt.Sprintf("/things/{%s}", ThingIDURIVar)
	protRoute.Get(thingPath, srv.handleRetrieveThing)
	protRoute.Put(thingPath, srv.handleUpdateThing)
	protRoute.Delete(thingPath, srv.handleDeleteThing)
	return srv
}

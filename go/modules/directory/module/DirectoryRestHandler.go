// Package api with the WoT defined REST API
package module

import (
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hivekit/go/modules/directory"
	"github.com/hiveot/hivekit/go/utils"
)

const ThingIDURIVar = "thingID"

// Handle directory HTTP requests for reading and writing the Thing directory as
// defined in the WoT discovery specification.
// This uses the given chi router which should have authentication/authorization
// middleware installed.
type DirectoryRestHandler struct {
	service directory.IDirectoryService
	router  *chi.Mux
}

func (srv *DirectoryRestHandler) handleRetrieveThing(w http.ResponseWriter, r *http.Request) {
	// A thingID is provided otherwise this handler would not have been called
	thingID := chi.URLParam(r, ThingIDURIVar)
	tdJSON, err := srv.service.RetrieveThing(thingID)
	utils.WriteReply(w, true, tdJSON, err)
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

// Input: TD document in JSON
// FIXME: implement access control; in the hub this TD must contain an ID with the agent prefix
// to indicate ownership.
func (srv *DirectoryRestHandler) handleUpdateThing(w http.ResponseWriter, r *http.Request) {
	var isAuthorized = false

	//clientID, err := GetClientIdFromContext(r)
	// isAuthorized = IsAdmin(clientID)

	if isAuthorized {
		tdJson, err := io.ReadAll(r.Body)
		if err == nil {
			err = srv.service.UpdateThing(string(tdJson))
		}
		utils.WriteReply(w, true, nil, err)
	} else {
		utils.WriteError(w, fmt.Errorf("not authorized to update the directory. Authorization not implemented"),
			http.StatusUnauthorized)
	}
}

// Create a new Directory REST handler and start listening on the given router
func StartDirectoryRestHandler(service directory.IDirectoryService, router *chi.Mux) *DirectoryRestHandler {
	srv := &DirectoryRestHandler{
		router:  router,
		service: service,
	}
	// add secured routes
	srv.router.Get(fmt.Sprintf("/things/{%s}", ThingIDURIVar), srv.handleRetrieveThing)
	srv.router.Get("/things", srv.handleRetrieveAllThings)
	srv.router.Put(fmt.Sprintf("/things/{%s}", ThingIDURIVar), srv.handleUpdateThing)
	return srv
}

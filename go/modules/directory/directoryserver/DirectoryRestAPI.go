// Package directoryserver with the WoT defined API
package directoryserver

import (
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hivekit/go/utils/net"
)

const ThingIDURIVar = "thingID"

// Serve HTTP requests for reading and writing the Thing directory as defined
// in the WoT discovery specification.
// All authenticated users can read the directory.
type DirectoryRestAPI struct {
	store  *DirectoryStore
	router *chi.Mux
}

// Input: TD document in JSON
// FIXME: implement access control; in the hub this TD must contain an ID with the agent prefix
// to indicate ownership.
func (srv *DirectoryRestAPI) handleUpdateThing(w http.ResponseWriter, r *http.Request) {
	var isAuthorized = false

	//clientID, err := GetClientIdFromContext(r)
	// isAuthorized = IsAdmin(clientID)

	if isAuthorized {
		tdJSON, err := io.ReadAll(r.Body)
		if err == nil {
			err = srv.store.UpdateThing(string(tdJSON))
		}
		net.WriteReply(w, true, nil, err)
	} else {
		net.WriteError(w, fmt.Errorf("not authorized to update the directory. Authorization not implemented"),
			http.StatusUnauthorized)
	}
}
func (srv *DirectoryRestAPI) handleRetrieveThing(w http.ResponseWriter, r *http.Request) {
	// A thingID is provided otherwise this handler would not have been called
	thingID := chi.URLParam(r, ThingIDURIVar)
	tdJSON, err := srv.store.RetrieveThing(thingID)
	net.WriteReply(w, true, tdJSON, err)
}

func (srv *DirectoryRestAPI) handleRetrieveAllThings(w http.ResponseWriter, r *http.Request) {
	qp := r.URL.Query()
	offsetStr := qp.Get("offset")
	limitStr := qp.Get("limit")
	offset, _ := strconv.ParseInt(offsetStr, 10, 32)
	limit, _ := strconv.ParseInt(limitStr, 10, 32)

	tdJSON, err := srv.store.RetrieveAllThings(int(offset), int(limit))
	net.WriteReply(w, true, tdJSON, err)
}

// Start the WoT API server and listen for http request
func (srv *DirectoryRestAPI) Start() {
	// add secured routes
	srv.router.Get(fmt.Sprintf("/things/{%s}", ThingIDURIVar), srv.handleRetrieveThing)
	srv.router.Get("/things", srv.handleRetrieveAllThings)
	srv.router.Put(fmt.Sprintf("/things/{%s}", ThingIDURIVar), srv.handleUpdateThing)
}

// Shutdown the server
func (srv *DirectoryRestAPI) Stop() {

}

func NewDirectoryRestAPI(store *DirectoryStore, router *chi.Mux) *DirectoryRestAPI {
	srv := &DirectoryRestAPI{
		router: router,
		store:  store,
	}
	return srv
}

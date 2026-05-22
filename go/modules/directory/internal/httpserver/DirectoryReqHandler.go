package internal

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules/directory"
	"github.com/hiveot/hivekit/go/utils"
)

// handleCreateThing creates a new TD in the directory
//
// Only agents and admin should be allowed to update the TD. This can be handled by authz.
// The thingID must contain the agent as the prefix to ensure unique namespace,
// so the stored ThingID will be agentID:thingID.
func (srv *DirectoryHttpServer) handleCreateThing(w http.ResponseWriter, r *http.Request) {
	var tdJson string
	rp, err := srv.httpServer.GetRequestParams(r)
	if err == nil {
		tdJson = string(rp.Payload)
		err = srv.Rpc(rp.ClientID, td.OpInvokeAction,
			srv.directoryThingID, directory.ActionCreateThing, tdJson, nil)
	}
	utils.WriteReply(w, true, nil, err) // 201
}

func (srv *DirectoryHttpServer) handleDeleteThing(w http.ResponseWriter, r *http.Request) {
	// A thingID is provided otherwise this handler would not have been called
	rp, err := srv.httpServer.GetRequestParams(r)
	thingID := chi.URLParam(r, ThingIDURIVar)

	req := msg.NewRequestMessage(rp.ClientID,
		td.OpInvokeAction, srv.directoryThingID, directory.ActionDeleteThing, thingID, "")
	_, err = srv.ForwardRequestWait(req)

	utils.WriteReply(w, true, nil, err) // 204
}

// Read the directory service TD itself
func (srv *DirectoryHttpServer) handleReadDirectoryTD(w http.ResponseWriter, r *http.Request) {
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
	utils.WriteReply(w, true, tdi, nil) // 200
}

func (srv *DirectoryHttpServer) handleRetrieveThing(w http.ResponseWriter, r *http.Request) {
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
	w.Write([]byte(tdJson)) // 200
}

func (srv *DirectoryHttpServer) handleRetrieveAllThings(w http.ResponseWriter, r *http.Request) {
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
	utils.WriteReply(w, true, tdList, err) // 200
}

// handleUpdateThing handle http request to update a Thing's TD
//
// Only agents and admin should be allowed to update the TD. This can be handled by authz.
// The thingID must contain the agent as the prefix to ensure unique namespace,
// so the stored ThingID will be agentID:thingID.
func (srv *DirectoryHttpServer) handleUpdateThing(w http.ResponseWriter, r *http.Request) {
	var tdJson string
	rp, err := srv.httpServer.GetRequestParams(r)
	if err == nil {
		tdJson = string(rp.Payload)
		err = srv.Rpc(rp.ClientID, td.OpInvokeAction,
			srv.directoryThingID, directory.ActionUpdateThing, tdJson, nil)
	}
	utils.WriteReply(w, true, nil, err) // 201
}

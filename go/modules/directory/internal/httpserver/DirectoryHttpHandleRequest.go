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

	rp, err := srv.httpServer.GetRequestParams(r)
	if err == nil {
		tdJson := string(rp.Payload) // ensure correct serialization of payload
		req := msg.NewRequestMessage(
			td.OpInvokeAction, srv.directoryThingID, directory.CreateThingAction, tdJson)
		req.SenderID = rp.ClientID
		_, err = srv.ForwardRequestWait(req)
	}
	utils.WriteReply(w, true, nil, err) // 201
}

func (srv *DirectoryHttpServer) handleDeleteThing(w http.ResponseWriter, r *http.Request) {
	// A thingID is provided otherwise this handler would not have been called
	rp, err := srv.httpServer.GetRequestParams(r)
	thingID := chi.URLParam(r, ThingIDURIVar)

	req := msg.NewRequestMessage(
		td.OpInvokeAction, srv.directoryThingID, directory.DeleteThingAction, thingID)
	req.SenderID = rp.ClientID
	_, err = srv.ForwardRequestWait(req)

	utils.WriteReply(w, true, nil, err) // 204
}

func (srv *DirectoryHttpServer) handleRetrieveThing(w http.ResponseWriter, r *http.Request) {
	var resp *msg.ResponseMessage
	// A thingID is provided otherwise this handler would not have been called
	thingID := chi.URLParam(r, ThingIDURIVar)
	rp, err := srv.httpServer.GetRequestParams(r)
	if err == nil {
		req := msg.NewRequestMessage(
			td.OpInvokeAction, srv.directoryThingID, directory.RetrieveThingAction, thingID)
		req.SenderID = rp.ClientID
		resp, err = srv.ForwardRequestWait(req)
	}
	if err != nil {
		utils.WriteError(w, err, 0)
		return
	}
	var tdocJson string
	err = resp.Decode(&tdocJson)
	w.Write([]byte(tdocJson)) // 200
}

func (srv *DirectoryHttpServer) handleRetrieveAllThings(w http.ResponseWriter, r *http.Request) {
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
		resp, err = srv.ForwardRequestWait(req)
	}
	utils.WriteReply(w, true, resp.Output, err) // 200
}

// handleUpdateThing handle http request to update a Thing's TD
//
// Only agents and admin should be allowed to update the TD. This can be handled by authz.
// The thingID must contain the agent as the prefix to ensure unique namespace,
// so the stored ThingID will be agentID:thingID.
func (srv *DirectoryHttpServer) handleUpdateThing(w http.ResponseWriter, r *http.Request) {
	var resp *msg.ResponseMessage
	rp, err := srv.httpServer.GetRequestParams(r)
	if err == nil {
		tdJson := string(rp.Payload) // ensure correct serialization of payload
		req := msg.NewRequestMessage(
			td.OpInvokeAction, srv.directoryThingID, directory.UpdateThingAction, tdJson)
		req.SenderID = rp.ClientID
		resp, err = srv.ForwardRequestWait(req)
		_ = resp
	}
	utils.WriteReply(w, true, nil, err) // 201
}

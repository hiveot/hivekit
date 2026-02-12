package server

import (
	"fmt"
	"log/slog"

	"github.com/araddon/dateparse"
	bucketserver "github.com/hiveot/hivekit/go/modules/bucketstore/server"
	"github.com/hiveot/hivekit/go/modules/history"
	"github.com/hiveot/hivekit/go/msg"
)

// ReadHistoryMsgHandler is the messaging request handler for reading the history
// This is a separate service thingID of the history service: ReadHistoryServiceID
// This support a cursor for long range iteration of the history.
type ReadHistoryMsgHandler struct {
	// routing address of the things to read history of
	histStore history.IHistoryModule

	// cache of remote cursors
	cursorCache *bucketserver.CursorCache

	isRunning bool
}

func (svc *ReadHistoryMsgHandler) _HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
	var resp *msg.ResponseMessage

	if req.SenderID == "" {
		return fmt.Errorf("missing senderID")
	}

	switch req.Name {
	case CreateCursorMethod:
		resp, err = svc.CreateCursor(req)
	case CursorFirstMethod:
		resp, err = svc.First(req)
	case CursorLastMethod:
		resp, err = svc.Last(req)
	case CursorNextMethod:
		resp, err = svc.Next(req)
	case CursorNextNMethod:
		resp, err = svc.NextN(req)
	case CursorPrevMethod:
		resp, err = svc.Prev(req)
	case CursorReleaseMethod:
		resp, err = svc.PrevN(req)
	case CursorSeekMethod:
		resp, err = svc.Seek(req)
	case ReadHistoryMethod:
		resp, err = svc.ReadHistory(req)
	}
	if err != nil {
		return err
	}
	return replyTo(resp)
}

// CreateCursor returns an iterator for ThingMessage objects.
func (svc *ReadHistoryMsgHandler) CreateCursor(req *msg.RequestMessage) (*msg.ResponseMessage, error) {
	var args CreateCursorArgs

	err := req.ToObject(&args)
	if args.ThingID == "" {
		return nil, fmt.Errorf("missing thingID")
	}
	slog.Info("CreateCursor for thing: ", "thingID", args.ThingID)

	cursorKey, err := svc.histStore.CreateCursor(req.SenderID, args.ThingID, args.Name)
	resp := req.CreateResponse(cursorKey, err)
	return resp, nil
}

// First
func (svc *ReadHistoryMsgHandler) First(req msg.RequestMessage) (*msg.ResponseMessage, error) {
	var cursorKey string
	var valueResp CursorValueResp

	err := req.ToObject(&cursorKey)
	if err != nil || cursorKey == "" {
		return nil, fmt.Errorf("missing cursorKey")
	}
	tv, valid, err := svc.histStore.First(req.SenderID, cursorKey)
	valueResp.Valid = valid
	valueResp.Value = tv
	resp := req.CreateResponse(valueResp, err)
	return resp, nil
}

// ReadHistory the history for the given time, duration and limit
// For more extensive result use the cursor
// To go back in time use the negative duration.
func (svc *ReadHistoryMsgHandler) ReadHistory(req msg.RequestMessage) (*msg.ResponseMessage, error) {
	var args ReadHistoryArgs
	var output ReadHistoryResp

	err := req.ToObject(&args)
	if err != nil || args.Timestamp == "" {
		return nil, fmt.Errorf("ReadHistory: Invalid arguments: " + err.Error())
	}
	ts, err := dateparse.ParseAny(args.Timestamp)
	if err != nil {
		return nil, fmt.Errorf("ReadHistory: Invalid timestamp: " + err.Error())
	}
	output.Values, output.ItemsRemaining, err = svc.histStore.ReadHistory(
		args.ThingID, args.AffordanceName, ts, args.Duration, args.Limit)
	resp := req.CreateActionResponse("", statuscompleted, output, err)
	return resp, err
}

// NewReadHistory starts the capability to read from a things's history
//
//	hc with the message bus connection. Its ID will be used as the agentID that provides the capability.
//	thingBucket is the open bucket used to store history data
func NewReadHistoryMsgHandler(histStore history.IHistoryModule) (svc *ReadHistoryMsgHandler) {

	svc = &ReadHistoryMsgHandler{
		histStore:   histStore,
		cursorCache: bucketserver.NewCursorCache(),
	}
	return svc
}

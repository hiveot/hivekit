package historyserver

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

// HandleRequest handles incoming requests for reading the history
func (svc *ReadHistoryMsgHandler) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
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
	case CursorPrevNMethod:
		resp, err = svc.PrevN(req)
	case CursorReleaseMethod:
		resp, err = svc.ReleaseCursor(req)
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
func (svc *ReadHistoryMsgHandler) First(req *msg.RequestMessage) (*msg.ResponseMessage, error) {
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

// Last
func (svc *ReadHistoryMsgHandler) Last(req *msg.RequestMessage) (*msg.ResponseMessage, error) {
	var cursorKey string
	var valueResp CursorValueResp

	err := req.ToObject(&cursorKey)
	if err != nil || cursorKey == "" {
		return nil, fmt.Errorf("missing cursorKey")
	}
	tv, valid, err := svc.histStore.Last(req.SenderID, cursorKey)
	valueResp.Valid = valid
	valueResp.Value = tv
	resp := req.CreateResponse(valueResp, err)
	return resp, nil
}

func (svc *ReadHistoryMsgHandler) Next(req *msg.RequestMessage) (*msg.ResponseMessage, error) {
	var cursorKey string
	var valueResp CursorValueResp

	err := req.ToObject(&cursorKey)
	if err != nil || cursorKey == "" {
		return nil, fmt.Errorf("missing cursorKey")
	}
	tv, valid, err := svc.histStore.Next(req.SenderID, cursorKey)
	valueResp.Valid = valid
	valueResp.Value = tv
	resp := req.CreateResponse(valueResp, err)
	return resp, nil
}

func (svc *ReadHistoryMsgHandler) NextN(req *msg.RequestMessage) (*msg.ResponseMessage, error) {
	var cursorNArgs CursorNArgs
	var cursorNResp CursorNResp

	err := req.ToObject(&cursorNArgs)
	if err != nil || cursorNArgs.CursorKey == "" {
		return nil, fmt.Errorf("missing cursorKey")
	}
	until, err := dateparse.ParseAny(cursorNArgs.Until)
	if err != nil {
		return nil, fmt.Errorf("invalid timestamp '%s': %s", cursorNArgs.Until, err.Error())
	}

	tvList, itemsRemaining, err := svc.histStore.NextN(
		req.SenderID, cursorNArgs.CursorKey, until, cursorNArgs.Limit)
	cursorNResp.Values = tvList
	cursorNResp.ItemsRemaining = itemsRemaining
	resp := req.CreateResponse(cursorNResp, err)
	return resp, nil
}

func (svc *ReadHistoryMsgHandler) Prev(req *msg.RequestMessage) (*msg.ResponseMessage, error) {
	var cursorKey string
	var valueResp CursorValueResp

	err := req.ToObject(&cursorKey)
	if err != nil || cursorKey == "" {
		return nil, fmt.Errorf("missing cursorKey")
	}
	tv, valid, err := svc.histStore.Prev(req.SenderID, cursorKey)
	valueResp.Valid = valid
	valueResp.Value = tv
	resp := req.CreateResponse(valueResp, err)
	return resp, nil
}

func (svc *ReadHistoryMsgHandler) PrevN(req *msg.RequestMessage) (*msg.ResponseMessage, error) {
	var cursorNArgs CursorNArgs
	var cursorNResp CursorNResp

	err := req.ToObject(&cursorNArgs)
	if err != nil || cursorNArgs.CursorKey == "" {
		return nil, fmt.Errorf("missing cursorKey")
	}
	until, err := dateparse.ParseAny(cursorNArgs.Until)
	if err != nil {
		return nil, fmt.Errorf("invalid timestamp '%s': %s", cursorNArgs.Until, err.Error())
	}
	tvList, itemsRemaining, err := svc.histStore.PrevN(
		req.SenderID, cursorNArgs.CursorKey, until, cursorNArgs.Limit)
	cursorNResp.Values = tvList
	cursorNResp.ItemsRemaining = itemsRemaining
	resp := req.CreateResponse(cursorNResp, err)
	return resp, nil
}

// ReadHistory the history for the given time, duration and limit
// For more extensive result use the cursor
// To go back in time use the negative duration.
func (svc *ReadHistoryMsgHandler) ReadHistory(req *msg.RequestMessage) (*msg.ResponseMessage, error) {
	var args ReadHistoryArgs
	var output ReadHistoryResp

	err := req.ToObject(&args)
	if err != nil || args.Timestamp == "" {
		return nil, fmt.Errorf("ReadHistory: Invalid arguments: " + err.Error())
	}
	ts, err := dateparse.ParseAny(args.Timestamp)
	if err != nil {
		return nil, fmt.Errorf("invalid timestamp '%s': %s", args.Timestamp, err.Error())
	}
	output.Values, output.ItemsRemaining, err = svc.histStore.ReadHistory(
		args.ThingID, args.AffordanceName, ts, args.Duration, args.Limit)
	resp := req.CreateResponse(output, err)
	return resp, nil
}

func (svc *ReadHistoryMsgHandler) ReleaseCursor(req *msg.RequestMessage) (*msg.ResponseMessage, error) {

	cursorKey := req.ToString(0)
	if cursorKey == "" {
		return nil, fmt.Errorf("missing cursorKey")
	}
	err := svc.histStore.ReleaseCursor(req.SenderID, cursorKey)
	resp := req.CreateResponse(nil, err)
	return resp, nil
}

func (svc *ReadHistoryMsgHandler) Seek(req *msg.RequestMessage) (*msg.ResponseMessage, error) {
	var seekArgs CursorSeekArgs
	var valueResp CursorValueResp

	err := req.ToObject(&seekArgs)
	if err != nil || seekArgs.CursorKey == "" {
		return nil, fmt.Errorf("missing cursorKey")
	}
	ts, err := dateparse.ParseAny(seekArgs.Timestamp)
	if err != nil {
		return nil, fmt.Errorf("invalid timestamp '%s': %s", seekArgs.Timestamp, err.Error())
	}
	tv, valid, err := svc.histStore.Seek(req.SenderID, seekArgs.CursorKey, ts)
	valueResp.Valid = valid
	valueResp.Value = tv
	resp := req.CreateResponse(valueResp, err)
	return resp, nil
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

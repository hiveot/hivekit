package internal

import (
	"fmt"
	"log/slog"

	"github.com/araddon/dateparse"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/modules/history"
)

// HistoryServiceImpl is the messaging request handler for reading the history.
// This support a cursor for long range iteration of the history.
// type HistoryServiceImpl struct {
// 	// routing address of the things to read history of
// 	histStore history.IHistoryService

// 	// cache of remote cursors
// 	cursorCache bucketstore.ICursorCache

// 	isRunning bool
// }

// HandleRequest handles incoming requests for reading the history
func (svc *HistoryServiceImpl) HandleRequest(req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
	var resp *msg.ResponseMessage

	if req.ThingID != svc.GetThingID() {
		// if the request is not for this module, store it and forward.
		go func() {
			if svc.config.RequestFilter.AcceptRequest(req) {
				svc.StoreRequest(req)
			}
		}()
		return svc.ForwardRequest(req, replyTo)
	} else {
		// handle requests for the history service itself

		if req.SenderID == "" {
			return fmt.Errorf("missing senderID")
		}

		switch req.Name {
		case history.CreateCursorMethod:
			resp, err = svc.handleCreateCursor(req)
		case history.CursorFirstMethod:
			resp, err = svc.handleFirst(req)
		case history.CursorLastMethod:
			resp, err = svc.handleLast(req)
		case history.CursorNextMethod:
			resp, err = svc.handleNext(req)
		case history.CursorNextNMethod:
			resp, err = svc.handleNextN(req)
		case history.CursorPrevMethod:
			resp, err = svc.handlePrev(req)
		case history.CursorPrevNMethod:
			resp, err = svc.handlePrevN(req)
		case history.CursorReleaseMethod:
			resp, err = svc.handleReleaseCursor(req)
		case history.CursorSeekMethod:
			resp, err = svc.handleSeek(req)
		case history.ReadHistoryMethod:
			resp, err = svc.handleReadHistory(req)
		}
		if err != nil {
			return err
		}
		return replyTo(resp)
	}
}

// handleCreateCursor returns an iterator for ThingMessage objects.
func (svc *HistoryServiceImpl) handleCreateCursor(req *msg.RequestMessage) (*msg.ResponseMessage, error) {
	var args history.CreateCursorArgs

	err := req.Decode(&args)
	if args.ThingID == "" {
		return nil, fmt.Errorf("missing thingID")
	}
	slog.Info("CreateCursor for thing: ", "thingID", args.ThingID)

	cursorKey, err := svc.CreateCursor(req.SenderID, args.ThingID, args.Name)
	resp := req.CreateResponse(cursorKey, err)
	return resp, nil
}

// handleFirst
func (svc *HistoryServiceImpl) handleFirst(req *msg.RequestMessage) (*msg.ResponseMessage, error) {
	var cursorKey string
	var valueResp history.CursorValueResp

	err := req.Decode(&cursorKey)
	if err != nil || cursorKey == "" {
		return nil, fmt.Errorf("missing cursorKey")
	}
	tv, valid, err := svc.First(req.SenderID, cursorKey)
	valueResp.Valid = valid
	valueResp.Value = tv
	resp := req.CreateResponse(valueResp, err)
	return resp, nil
}

// handleLast
func (svc *HistoryServiceImpl) handleLast(req *msg.RequestMessage) (*msg.ResponseMessage, error) {
	var cursorKey string
	var valueResp history.CursorValueResp

	err := req.Decode(&cursorKey)
	if err != nil || cursorKey == "" {
		return nil, fmt.Errorf("missing cursorKey")
	}
	tv, valid, err := svc.Last(req.SenderID, cursorKey)
	valueResp.Valid = valid
	valueResp.Value = tv
	resp := req.CreateResponse(valueResp, err)
	return resp, nil
}

func (svc *HistoryServiceImpl) handleNext(req *msg.RequestMessage) (*msg.ResponseMessage, error) {
	var cursorKey string
	var valueResp history.CursorValueResp

	err := req.Decode(&cursorKey)
	if err != nil || cursorKey == "" {
		return nil, fmt.Errorf("missing cursorKey")
	}
	tv, valid, err := svc.Next(req.SenderID, cursorKey)
	valueResp.Valid = valid
	valueResp.Value = tv
	resp := req.CreateResponse(valueResp, err)
	return resp, nil
}

func (svc *HistoryServiceImpl) handleNextN(req *msg.RequestMessage) (*msg.ResponseMessage, error) {
	var cursorNArgs history.CursorNArgs
	var cursorNResp history.CursorNResp

	err := req.Decode(&cursorNArgs)
	if err != nil || cursorNArgs.CursorKey == "" {
		return nil, fmt.Errorf("missing cursorKey")
	}
	until, err := dateparse.ParseAny(cursorNArgs.Until)
	if err != nil {
		return nil, fmt.Errorf("invalid timestamp '%s': %s", cursorNArgs.Until, err.Error())
	}

	tvList, itemsRemaining, err := svc.NextN(
		req.SenderID, cursorNArgs.CursorKey, until, cursorNArgs.Limit)
	cursorNResp.Values = tvList
	cursorNResp.ItemsRemaining = itemsRemaining
	resp := req.CreateResponse(cursorNResp, err)
	return resp, nil
}

func (svc *HistoryServiceImpl) handlePrev(req *msg.RequestMessage) (*msg.ResponseMessage, error) {
	var cursorKey string
	var valueResp history.CursorValueResp

	err := req.Decode(&cursorKey)
	if err != nil || cursorKey == "" {
		return nil, fmt.Errorf("missing cursorKey")
	}
	tv, valid, err := svc.Prev(req.SenderID, cursorKey)
	valueResp.Valid = valid
	valueResp.Value = tv
	resp := req.CreateResponse(valueResp, err)
	return resp, nil
}

func (svc *HistoryServiceImpl) handlePrevN(req *msg.RequestMessage) (*msg.ResponseMessage, error) {
	var cursorNArgs history.CursorNArgs
	var cursorNResp history.CursorNResp

	err := req.Decode(&cursorNArgs)
	if err != nil || cursorNArgs.CursorKey == "" {
		return nil, fmt.Errorf("missing cursorKey")
	}
	until, err := dateparse.ParseAny(cursorNArgs.Until)
	if err != nil {
		return nil, fmt.Errorf("invalid timestamp '%s': %s", cursorNArgs.Until, err.Error())
	}
	tvList, itemsRemaining, err := svc.PrevN(
		req.SenderID, cursorNArgs.CursorKey, until, cursorNArgs.Limit)
	cursorNResp.Values = tvList
	cursorNResp.ItemsRemaining = itemsRemaining
	resp := req.CreateResponse(cursorNResp, err)
	return resp, nil
}

// handleReadHistory the history for the given time, duration and limit
// For more extensive result use the cursor
// To go back in time use the negative duration.
func (svc *HistoryServiceImpl) handleReadHistory(req *msg.RequestMessage) (*msg.ResponseMessage, error) {
	var args history.ReadHistoryArgs
	var output history.ReadHistoryResp

	err := req.Decode(&args)
	if err != nil || args.Timestamp == "" {
		return nil, fmt.Errorf("ReadHistory: Invalid arguments: %w", err)
	}
	ts, err := dateparse.ParseAny(args.Timestamp)
	if err != nil {
		return nil, fmt.Errorf("invalid timestamp '%s': %s", args.Timestamp, err.Error())
	}
	output.Values, output.ItemsRemaining, err = svc.ReadHistory(
		args.ThingID, args.AffordanceName, ts, args.Duration, args.Limit)
	resp := req.CreateResponse(output, err)
	return resp, nil
}

func (svc *HistoryServiceImpl) handleReleaseCursor(req *msg.RequestMessage) (*msg.ResponseMessage, error) {

	cursorKey := req.ToString(0)
	if cursorKey == "" {
		return nil, fmt.Errorf("missing cursorKey")
	}
	err := svc.ReleaseCursor(req.SenderID, cursorKey)
	resp := req.CreateResponse(nil, err)
	return resp, nil
}

func (svc *HistoryServiceImpl) handleSeek(req *msg.RequestMessage) (*msg.ResponseMessage, error) {
	var seekArgs history.CursorSeekArgs
	var valueResp history.CursorValueResp

	err := req.Decode(&seekArgs)
	if err != nil || seekArgs.CursorKey == "" {
		return nil, fmt.Errorf("missing cursorKey")
	}
	ts, err := dateparse.ParseAny(seekArgs.Timestamp)
	if err != nil {
		return nil, fmt.Errorf("invalid timestamp '%s': %s", seekArgs.Timestamp, err.Error())
	}
	tv, valid, err := svc.Seek(req.SenderID, seekArgs.CursorKey, ts)
	valueResp.Valid = valid
	valueResp.Value = tv
	resp := req.CreateResponse(valueResp, err)
	return resp, nil
}

// NewReadHistory starts the capability to read from a things's history
//
//	hc with the message bus connection. Its ID will be used as the agentID that provides the capability.
//	thingBucket is the open bucket used to store history data
// func NewHistoryServiceImpl(histStore history.IHistoryService) (svc *HistoryServiceImpl) {

// 	svc = &HistoryServiceImpl{
// 		histStore:   histStore,
// 		cursorCache: bucketstorepkg.NewCursorCache(),
// 	}
// 	return svc
// }

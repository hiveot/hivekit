package api

import (
	_ "embed"
	"fmt"
	"log/slog"
	"time"

	"github.com/hiveot/hivekit/go/modules/messaging"
	"github.com/hiveot/hivekit/go/modules/services/bucketstore"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hivekit/go/wot"
)

// Embed the store TM
//
//go:embed bucketstore-tm.json
var BucketStoreTMJson []byte

// DirectoryMsgHandler maps SME messages to the native directory interface
type BucketMsgHandler struct {
	// thingID of this instance
	thingID string
	// the underlying bucket store to access
	service bucketstore.IBucketStore
	// serving cursor requests
	cursorCache *bucketstore.CursorCache
}

// HandleRequest handles action requests for the service
// This returns nil if thingID, operation or request name is not recognized.
// If the request is missing a senderID then an error is returned.
func (handler *BucketMsgHandler) HandleRequest(req *messaging.RequestMessage) *messaging.ResponseMessage {
	// TODO: should this verify the destination this instance with an instance thingID?
	if req.ThingID != handler.thingID {
		return nil
	} else if req.SenderID == "" {
		err := fmt.Errorf("missing senderID in request")
		return req.CreateErrorResponse(err)
	}
	if req.Operation == wot.OpInvokeAction {
		switch req.Name {
		case ActionDelete:
			return handler.Delete(req)
		case ActionGet:
			return handler.Get(req)
		case ActionGetMultiple:
			return handler.GetMultiple(req)
		case ActionSet:
			return handler.Set(req)
		case ActionSetMultiple:
			return handler.SetMultiple(req)
		}
	}
	err := fmt.Errorf("unknown action '%s' for service '%s'", req.Name, req.ThingID)
	resp := req.CreateResponse(nil, err)
	return resp
}

// Cursor returns an iterator for objects.
// The cursor expires one minute after it is last used.
// This returns a cursor ID that can be used in the first,last,next,prev methods
func (handler *BucketMsgHandler) Cursor(req *messaging.RequestMessage) *messaging.ResponseMessage {
	var err error
	lifespan := time.Minute
	if req.ThingID == "" {
		err = fmt.Errorf("missing thingID")
		return req.CreateErrorResponse(err)
	}
	slog.Info("GetCursor for bucket: ", "senderID", req.SenderID)
	bucket := handler.service.GetBucket(req.SenderID)
	cursor, err := bucket.Cursor()
	//
	if err != nil {
		return req.CreateErrorResponse(err)
	}
	cursorKey := handler.cursorCache.Add(cursor, bucket, req.SenderID, lifespan)
	resp := req.CreateResponse(cursorKey, nil)
	return resp
}

func (handler *BucketMsgHandler) Delete(req *messaging.RequestMessage) *messaging.ResponseMessage {
	var objectKey string
	// use the bucket of the authenticated sender
	bucket := handler.service.GetBucket(req.SenderID)
	err := utils.Decode(req.Input, &objectKey)
	if err == nil {
		err = bucket.Delete(objectKey)
	}
	return req.CreateResponse(nil, err)
}

func (handler *BucketMsgHandler) Get(req *messaging.RequestMessage) *messaging.ResponseMessage {
	var objectKey string
	var raw []byte
	bucket := handler.service.GetBucket(req.SenderID)
	err := utils.Decode(req.Input, &objectKey)
	if err == nil {
		raw, err = bucket.Get(objectKey)
	}
	if err != nil {
		return req.CreateErrorResponse(err)
	}
	return req.CreateResponse(string(raw), nil)
}

func (handler *BucketMsgHandler) GetMultiple(req *messaging.RequestMessage) *messaging.ResponseMessage {
	var docKeys []string
	var raw map[string][]byte = nil
	result := make(map[string]string)

	bucket := handler.service.GetBucket(req.SenderID)
	err := utils.Decode(req.Input, &docKeys)
	if err == nil {
		raw, err = bucket.GetMultiple(docKeys)
	}
	if err == nil {
		for k, v := range raw {
			result[k] = string(v)
		}
	}
	return req.CreateResponse(result, err)
}

func (handler *BucketMsgHandler) Set(req *messaging.RequestMessage) *messaging.ResponseMessage {
	bucket := handler.service.GetBucket(req.SenderID)
	input := SetArgs{}
	err := utils.Decode(req.Input, &input)
	if err == nil {
		err = bucket.Set(input.Key, []byte(input.Doc))
	}
	return req.CreateResponse(nil, err)
}

func (handler *BucketMsgHandler) SetMultiple(req *messaging.RequestMessage) *messaging.ResponseMessage {
	bucket := handler.service.GetBucket(req.SenderID)
	var input map[string]string
	raw := make(map[string][]byte)
	err := utils.Decode(req.Input, &input)
	if err == nil {
		for k, v := range input {
			raw[k] = []byte(v)
		}
		err = bucket.SetMultiple(raw)
	}
	return req.CreateResponse(nil, err)
}

// NewBucketMsgHandler returns a new instance of the messaging handler for
// serving bucket requests.
// This opens buckets using the authenticated client's senderID.
func NewBucketMsgHandler(thingID string, service bucketstore.IBucketStore) *BucketMsgHandler {
	agent := BucketMsgHandler{
		thingID:     thingID,
		service:     service,
		cursorCache: bucketstore.NewCursorCache(),
	}
	return &agent
}

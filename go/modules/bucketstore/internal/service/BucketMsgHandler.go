package service

import (
	_ "embed"
	"fmt"
	"log/slog"
	"time"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	bucketstore "github.com/hiveot/hivekit/go/modules/bucketstore"
	"github.com/hiveot/hivekit/go/utils"
)

// Embed the store TM
//
//go:embed bucketstore-tm.json
var BucketStoreTMJson []byte

// DirectoryMsgHandler maps RRN messages to the native directory interface
// This uses the authenticated sender clientID as the bucket ID for requests.
type BucketMsgHandler struct {
	// thingID of this instance
	thingID string
	// the underlying bucket store to access
	service bucketstore.IBucketStore
	// serving cursor requests
	cursorCache *CursorCache
}

// HandleRequest handles action requests for the service
// This returns nil if thingID, operation or request name is not recognized.
// If the request is missing a senderID then an error is returned.
func (svc *BucketStoreService) handleBucketStoreRequest(
	req *msg.RequestMessage, replyTo msg.ResponseHandler) error {

	var resp *msg.ResponseMessage
	// the senderID is required to select the bucket
	if req.SenderID == "" {
		return fmt.Errorf("missing senderID in request")
	}
	if req.Operation == td.OpInvokeAction {
		switch req.Name {
		case bucketstore.ActionDelete:
			resp = svc.Delete(req)
		case bucketstore.ActionGet:
			resp = svc.Get(req)
		case bucketstore.ActionGetMultiple:
			resp = svc.GetMultiple(req)
		case bucketstore.ActionSet:
			resp = svc.Set(req)
		case bucketstore.ActionSetMultiple:
			resp = svc.SetMultiple(req)

		default:
			return fmt.Errorf("unknown action '%s' for service '%s'", req.Name, req.ThingID)
		}
	}
	if resp == nil {
		return fmt.Errorf("handleBucketStoreRequest: Unexpected request: op=%s, name=%s",
			req.Operation, req.Name)
	}
	return replyTo(resp)
}

// Cursor returns an iterator for objects.
// The cursor expires one minute after it is last used.
// This returns a cursor ID that can be used in the first,last,next,prev methods
func (svc *BucketStoreService) Cursor(req *msg.RequestMessage) *msg.ResponseMessage {
	var err error
	lifespan := time.Minute
	if req.ThingID == "" {
		err = fmt.Errorf("missing thingID")
		return req.CreateErrorResponse(err)
	}
	slog.Info("GetCursor for bucket: ", "senderID", req.SenderID)
	bucket := svc.store.GetBucket(req.SenderID)
	cursor, err := bucket.Cursor()
	//
	if err != nil {
		return req.CreateErrorResponse(err)
	}
	cursorKey := svc.cursorCache.Add(req.SenderID, cursor, bucket, "", lifespan)
	resp := req.CreateResponse(cursorKey, nil)
	return resp
}

func (svc *BucketStoreService) Delete(req *msg.RequestMessage) *msg.ResponseMessage {
	var objectKey string
	// use the bucket of the authenticated sender
	bucket := svc.store.GetBucket(req.SenderID)
	err := utils.Decode(req.Input, &objectKey)
	if err == nil {
		err = bucket.Delete(objectKey)
	}
	return req.CreateResponse(nil, err)
}

func (svc *BucketStoreService) Get(req *msg.RequestMessage) *msg.ResponseMessage {
	var objectKey string
	var raw []byte
	bucket := svc.store.GetBucket(req.SenderID)
	err := utils.Decode(req.Input, &objectKey)
	if err == nil {
		raw, err = bucket.Get(objectKey)
	}
	if err != nil {
		return req.CreateErrorResponse(err)
	}
	return req.CreateResponse(string(raw), nil)
}

func (svc *BucketStoreService) GetMultiple(req *msg.RequestMessage) *msg.ResponseMessage {
	var docKeys []string
	var raw map[string][]byte = nil
	result := make(map[string]string)

	bucket := svc.store.GetBucket(req.SenderID)
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

func (svc *BucketStoreService) Set(req *msg.RequestMessage) *msg.ResponseMessage {
	bucket := svc.store.GetBucket(req.SenderID)
	input := bucketstore.SetArgs{}
	err := utils.Decode(req.Input, &input)
	if err == nil {
		err = bucket.Set(input.Key, []byte(input.Doc))
	}
	return req.CreateResponse(nil, err)
}

func (svc *BucketStoreService) SetMultiple(req *msg.RequestMessage) *msg.ResponseMessage {
	bucket := svc.store.GetBucket(req.SenderID)
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

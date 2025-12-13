package api

import (
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/messaging"
	"github.com/hiveot/hivekit/go/wot"
)

// BucketMsgClient is the SME client for service messages using a provided hub connection.
//
// The BucketMsgClient converts the bucketstore API to SME messages and passes them
// to the provided sink, typically a messaging protocol client.
//
// It can be used to call a bucket store instance remotely using any of the protocols
// supporting SME.
//
// This requires that the client is authenticated and uses the client ID as the
// bucket name.
//
// This implements the IBucket interface.
type BucketMsgClient struct {
	// thingID ID of the storage service instance
	thingID string // bucket store service instance ID
	sink    modules.IHiveModule
}

// Delete removes the record with the given key.
func (cl *BucketMsgClient) Delete(key string) error {
	req := messaging.NewRequestMessage(
		wot.OpInvokeAction, cl.thingID, ActionDelete, key, "")
	resp := cl.sink.HandleRequest(req)
	return resp.AsError()
}

// Get reads the record with the given key.
// If the key doesn't exist this returns an error.
func (cl *BucketMsgClient) Get(key string) (doc string, err error) {

	req := messaging.NewRequestMessage(wot.OpInvokeAction, cl.thingID, ActionGet, key, "")
	resp := cl.sink.HandleRequest(req)

	err = resp.Decode(&doc)
	if err != nil {
		return "", err
	}
	return doc, err
}

// GetMultiple reads multiple serialized records with the given keys.
func (cl *BucketMsgClient) GetMultiple(keys []string) (values map[string]string, err error) {

	req := messaging.NewRequestMessage(
		wot.OpInvokeAction, cl.thingID, ActionGetMultiple, keys, "")
	resp := cl.sink.HandleRequest(req)
	err = resp.Decode(&values)
	return values, err
}

// Set serializes and stores a record by the given key
func (cl *BucketMsgClient) Set(key string, doc string) error {
	args := SetArgs{
		Key: key,
		Doc: doc,
	}
	req := messaging.NewRequestMessage(
		wot.OpInvokeAction, cl.thingID, ActionSet, args, "")
	resp := cl.sink.HandleRequest(req)
	err := resp.AsError()
	return err
}

// SetMultiple writes multiple serialized records
func (cl *BucketMsgClient) SetMultiple(kv map[string]string) error {
	args := make(map[string]string)
	for k, v := range kv {
		args[k] = v
	}
	req := messaging.NewRequestMessage(
		wot.OpInvokeAction, cl.thingID, ActionSetMultiple, args, "")
	resp := cl.sink.HandleRequest(req)
	err := resp.AsError()
	return err
}

// NewBucketStoreMsgClient returns a client to access the bucket store.
// The sink is the handler for message delivery.
//
//	thingID is the unique ID of the bucket store instance
//	sink is the handler of request messages
func NewBucketStoreMsgClient(thingID string, sink modules.IHiveModule) *BucketMsgClient {
	cl := BucketMsgClient{
		thingID: thingID,
		sink:    sink,
	}
	return &cl
}

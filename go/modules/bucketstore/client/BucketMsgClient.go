package bucketstoreclient

import (
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/bucketstore"
	"github.com/hiveot/hivekit/go/msg"
	"github.com/hiveot/hivekit/go/wot"
)

// BucketMsgClient is the RRN client for service messages using a provided hub connection.
//
// The BucketMsgClient converts the bucketstore API to RRN messages and passes them
// to the provided sink, typically a messaging protocol client.
//
// It can be used to call a bucket store instance remotely using any of the protocols
// supporting RRN messaging.
//
// This requires that the client is authenticated and uses the client ID as the
// bucket name.
//
// This implements the IBucket interface.
type BucketMsgClient struct {
	modules.HiveModuleBase
	// thingID ID of the storage service instance
	thingID string // bucket store service instance ID

}

// Delete removes the record with the given key.
func (cl *BucketMsgClient) Delete(key string) error {
	req := msg.NewRequestMessage(
		wot.OpInvokeAction, cl.thingID, bucketstore.ActionDelete, key, "")
	_, err := cl.ForwardRequestWait(req)
	return err
}

// Get reads the record with the given key.
// If the key doesn't exist this returns an error.
func (cl *BucketMsgClient) Get(key string) (doc string, err error) {

	req := msg.NewRequestMessage(wot.OpInvokeAction, cl.thingID, bucketstore.ActionGet, key, "")
	resp, err := cl.ForwardRequestWait(req)

	err = resp.Decode(&doc)
	if err != nil {
		return "", err
	}
	return doc, err
}

// GetMultiple reads multiple serialized records with the given keys.
func (cl *BucketMsgClient) GetMultiple(keys []string) (values map[string]string, err error) {

	req := msg.NewRequestMessage(
		wot.OpInvokeAction, cl.thingID, bucketstore.ActionGetMultiple, keys, "")
	resp, err := cl.ForwardRequestWait(req)
	err = resp.Decode(&values)
	return values, err
}

// Set serializes and stores a record by the given key
func (cl *BucketMsgClient) Set(key string, doc string) error {
	args := bucketstore.SetArgs{
		Key: key,
		Doc: doc,
	}
	req := msg.NewRequestMessage(
		wot.OpInvokeAction, cl.thingID, bucketstore.ActionSet, args, "")
	_, err := cl.ForwardRequestWait(req)
	return err
}

// SetMultiple writes multiple serialized records
func (cl *BucketMsgClient) SetMultiple(kv map[string]string) error {
	args := make(map[string]string)
	for k, v := range kv {
		args[k] = v
	}
	req := msg.NewRequestMessage(
		wot.OpInvokeAction, cl.thingID, bucketstore.ActionSetMultiple, args, "")
	_, err := cl.ForwardRequestWait(req)
	return err
}

// NewBucketStoreMsgClient returns a client to access the bucket store.
// Use the sink to attach a transport module.
//
//	thingID is the instance ID of the bucket store module
//	sink is the handler that forwards messages to the module. Typically a messaging client.
func NewBucketStoreMsgClient(thingID string, sink modules.IHiveModule) *BucketMsgClient {
	cl := BucketMsgClient{
		thingID: thingID,
	}
	cl.Init(thingID+"-client", sink)
	return &cl
}

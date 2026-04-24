package bucketstorepkg

import (
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules"
	"github.com/hiveot/hivekit/go/modules/bucketstore"
)

// BucketMsgClient is the client for using a remote bucket store server module.
//
// The BucketMsgClient converts the bucketstore API to RRN messages and passes them
// to the provided sink, typically a messaging protocol client.
type BucketMsgClient struct {
	modules.HiveModuleBase
	// storeThingID ID of the storage service instance
	storeThingID string // bucket store service instance ID
}

// Close ends the use of this client and frees its resources
func (cl *BucketMsgClient) Close() error {
	return nil
}

// Delete removes the record with the given key.
func (cl *BucketMsgClient) Delete(key string) error {
	err := cl.Rpc("", td.OpInvokeAction, cl.storeThingID, bucketstore.ActionDelete, key, nil)
	return err
}

// Get reads the record with the given key.
// If the key doesn't exist this returns an error.
func (cl *BucketMsgClient) Get(key string) (doc string, err error) {
	err = cl.Rpc("", td.OpInvokeAction, cl.storeThingID, bucketstore.ActionGet, key, &doc)
	return doc, err
}

// GetMultiple reads multiple serialized records with the given keys.
func (cl *BucketMsgClient) GetMultiple(keys []string) (values map[string]string, err error) {
	err = cl.Rpc("", td.OpInvokeAction,
		cl.storeThingID, bucketstore.ActionGetMultiple, keys, &values)
	return values, err
}

// Set serializes and stores a record by the given key
func (cl *BucketMsgClient) Set(key string, doc string) error {
	args := bucketstore.SetArgs{
		Key: key,
		Doc: doc,
	}
	err := cl.Rpc("", td.OpInvokeAction,
		cl.storeThingID, bucketstore.ActionSet, args, nil)
	return err
}

// SetMultiple writes multiple serialized records
func (cl *BucketMsgClient) SetMultiple(kv map[string]string) error {
	args := make(map[string]string)
	for k, v := range kv {
		args[k] = v
	}
	err := cl.Rpc("", td.OpInvokeAction,
		cl.storeThingID, bucketstore.ActionSetMultiple, args, nil)
	return err
}

// NewBucketStoreMsgClient returns a client module to access a remote bucket store.
// Use the sink to attach a module connected to a transport.
//
//	thingID is the instance ID of the bucket store module
//	sink is the handler that forwards messages to the module. Typically a messaging client.
func NewBucketStoreMsgClient(thingID string, sink modules.IHiveModule) *BucketMsgClient {
	cl := &BucketMsgClient{
		storeThingID: thingID,
	}
	cl.SetRequestSink(sink.HandleRequest)
	sink.SetNotificationSink(cl.HandleNotification)

	return cl
}

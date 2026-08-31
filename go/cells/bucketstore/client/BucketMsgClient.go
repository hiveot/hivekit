package bucketstore_client

import (
	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/cells"
	"github.com/hiveot/hivekit/go/cells/bucketstore"
)

// BucketMsgClient is the client for using a remote bucket store service.
//
// The BucketMsgClient converts the bucketstore API to RRN messages and passes them
// to the provided sink, typically a messaging protocol client.
type BucketMsgClient struct {
	*cells.HiveCellBase
	// storeServiceID ID of the storage service instance
	storeServiceID string // bucket store service instance ID
}

// Close ends the use of this client and frees its resources
func (cl *BucketMsgClient) Close() error {
	return nil
}

// Delete removes the record with the given key.
func (cl *BucketMsgClient) Delete(key string) error {
	err := cl.Rpc(td.OpInvokeAction, cl.storeServiceID, bucketstore.ActionDelete, key, nil)
	return err
}

// Get reads the record with the given key.
// If the key doesn't exist this returns an error.
func (cl *BucketMsgClient) Get(key string) (doc string, err error) {
	err = cl.Rpc(td.OpInvokeAction, cl.storeServiceID, bucketstore.ActionGet, key, &doc)
	return doc, err
}

// GetMultiple reads multiple serialized records with the given keys.
func (cl *BucketMsgClient) GetMultiple(keys []string) (values map[string]string, err error) {
	err = cl.Rpc(td.OpInvokeAction,
		cl.storeServiceID, bucketstore.ActionGetMultiple, keys, &values)
	return values, err
}

// Set serializes and stores a record by the given key
func (cl *BucketMsgClient) Set(key string, doc string) error {
	args := bucketstore.SetArgs{
		Key: key,
		Doc: doc,
	}
	err := cl.Rpc(td.OpInvokeAction,
		cl.storeServiceID, bucketstore.ActionSet, args, nil)
	return err
}

// SetMultiple writes multiple serialized records
func (cl *BucketMsgClient) SetMultiple(kv map[string]string) error {
	args := make(map[string]string)
	for k, v := range kv {
		args[k] = v
	}
	err := cl.Rpc(td.OpInvokeAction,
		cl.storeServiceID, bucketstore.ActionSetMultiple, args, nil)
	return err
}

// StartBucketStoreMsgClient starts and links a client to access a remote bucket store.
// Use the sink to attach the client to a transport cell.
//
//	sink is the cell that forwards requests. nil to do this manually.
//	serviceID is the cell thingID of the remote bucket store service
func StartBucketStoreMsgClient(sink api.IHiveCell, serviceID string) *BucketMsgClient {
	cl := &BucketMsgClient{
		storeServiceID: serviceID,
		HiveCellBase:   cells.NewHiveCellBase("", 0),
	}
	// link to the sink.
	if sink != nil {
		cl.SetRequestSink(sink)
		sink.SetNotificationSink(cl)
	}
	return cl
}

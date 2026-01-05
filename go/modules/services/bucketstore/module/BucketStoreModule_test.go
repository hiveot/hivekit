package module_test

import (
	"testing"

	"github.com/hiveot/hivekit/go/modules/services/bucketstore/api"
	"github.com/hiveot/hivekit/go/modules/services/bucketstore/module"
	"github.com/hiveot/hivekit/go/modules/transports/direct"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// use in-memory storage
var storageRoot = ""

func startModule(t *testing.T) (*module.BucketStoreModule, func(), error) {
	m := module.NewBucketStoreModule(storageRoot)
	err := m.Start()
	require.NoError(t, err)
	return m, func() {
		m.Stop()
	}, err
}

// Generic store store testcases
func TestStartStop(t *testing.T) {
	t.Logf("---%s---\n", t.Name())

	m, stopFn, err := startModule(t)
	_ = m
	require.NoError(t, err)
	defer stopFn()
}

func TestGetSetMsgAPI(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	clientID := "client1"
	key1 := "key1"
	key2 := "key2"
	key3 := "key3"
	val1 := "value1"
	val2 := ""
	val3 := "value3"

	m, stopFn, err := startModule(t)
	require.NoError(t, err)
	defer stopFn()
	tp := direct.NewDirectTransport(clientID, nil, m)
	cl := api.NewBucketStoreMsgClient(m.GetModuleID(), tp)
	err = cl.Set(key1, val1)
	require.NoError(t, err)

	doc1, err := cl.Get(key1)
	require.NoError(t, err)
	assert.Equal(t, val1, doc1)

	// expect only a single result
	keys := []string{key1, key2}
	docs, err := cl.GetMultiple(keys)
	require.NoError(t, err)
	require.Len(t, docs, 1)
	assert.Equal(t, val1, docs[key1])

	// add second and third keys and expect 2 results
	kvmap2 := map[string]string{
		key2: val2,
		key3: val3,
	}
	err = cl.SetMultiple(kvmap2)
	require.NoError(t, err)

	keys = []string{key1, key2, key3}
	docs, err = cl.GetMultiple(keys)
	require.NoError(t, err)
	require.Len(t, docs, 3)
	assert.Equal(t, val2, docs[key2])

	// last, delete 1st
	err = cl.Delete(key1)
	require.NoError(t, err)
	val1b, err := cl.Get(key1)
	assert.Error(t, err)
	assert.Empty(t, val1b)

}

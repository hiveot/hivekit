package module_test

import (
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hivekit/go/modules/services/bucketstore"
	"github.com/hiveot/hivekit/go/modules/services/bucketstore/kvbtree"
	"github.com/hiveot/hivekit/go/modules/services/directory"
	"github.com/hiveot/hivekit/go/modules/services/directory/api"
	"github.com/hiveot/hivekit/go/modules/services/directory/module"
	"github.com/hiveot/hivekit/go/modules/transports/direct"
	"github.com/hiveot/hivekit/go/wot/td"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var storageRoot = ""

// Generic directory store testcases
func TestStartStop(t *testing.T) {
	t.Logf("---%s---\n", t.Name())

	var bucketStore bucketstore.IBucketStore = kvbtree.NewKVStore("") // in-memory only
	err := bucketStore.Open()
	require.NoError(t, err)
	defer bucketStore.Close()
	router := chi.NewRouter()

	m := module.NewDirectoryModule(storageRoot, router)
	err = m.Start()
	require.NoError(t, err)
	defer m.Stop()

	// add a thing
	tdJson := api.DirectoryTMJson
	svc := m.GetService()
	require.NotNil(t, svc)
	svc.UpdateThing(string(tdJson))

	// read all things
	tdList, err := svc.RetrieveAllThings(0, 10)
	assert.NoError(t, err)
	assert.NotNil(t, tdList)
}

func TestCRUDUsingMsgAPI(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	const clientID = "user1"

	directoryID := directory.DefaultDirectoryThingID
	thing1ID := "thing1"

	var bucketStore bucketstore.IBucketStore = kvbtree.NewKVStore("") // in-memory only
	err := bucketStore.Open()
	assert.NoError(t, err)
	router := chi.NewRouter()

	m := module.NewDirectoryModule(storageRoot, router)
	err = m.Start()
	require.NoError(t, err)
	defer m.Stop()

	// test create a TD
	tdi1 := td.NewTD(thing1ID, "thing 1", "device")
	tdi1Json := tdi1.ToString()

	// use a direct transport to the directory as the sink for the client
	tp := direct.NewDirectTransport(clientID, nil, m)
	dirClient := api.NewDirectoryMsgClient(directoryID, tp)
	err = dirClient.CreateThing(tdi1Json)
	require.NoError(t, err)

	// read the new TD
	tdi2Json, err := dirClient.RetrieveThing(thing1ID)
	require.NoError(t, err)
	tdi2, err := td.UnmarshalTD(tdi2Json)
	require.NoError(t, err)
	assert.Equal(t, thing1ID, tdi2.ID)

	// delete the new TD
	err = dirClient.DeleteThing(thing1ID)
	require.NoError(t, err)

	// read should fail
	_, err = dirClient.RetrieveThing(thing1ID)
	require.Error(t, err)
}

// the rest api needs a http server module
// func TestCRUDUsingRestAPI(t *testing.T) {
// 	t.Logf("---%s---\n", t.Name())

// 	// directoryID := module.DefaultDirectoryThingID
// 	thing1ID := "thing1"

// 	var bucketStore bucketstore.IBucketStore = kvbtree.NewKVStore("") // in-memory only
// 	err := bucketStore.Open()
// 	require.NoError(t, err)
// 	defer bucketStore.Close()
// 	router := chi.NewRouter()

// 	m := module.NewDirectoryModule(bucketStore, router)
// 	err = m.Start("")
// 	require.NoError(t, err)
// 	defer m.Stop()

// 	// connect the client to the server
// 	tddUrl := fmt.Sprintf("https://localhost:%d", port)

// 	msgClient := api.NewDirectoryRestClient(0)
// 	err = msgClient.Connect(tddUrl, clientID, authToken, caCert)
// 	require.NoError(t, err)

// 	// test create a TD
// 	tdi1 := td.NewTD(thing1ID, "thing 1", "device")
// 	tdi1Json := tdi1.ToString()

// 	err = msgClient.CreateThing(tdi1Json)
// 	require.NoError(t, err)

// 	// read the new TD
// 	tdi2Json, err := msgClient.RetrieveThing(thing1ID)
// 	require.NoError(t, err)
// 	tdi2, err := td.UnmarshalTD(tdi2Json)
// 	require.NoError(t, err)
// 	assert.Equal(t, thing1ID, tdi2.ID)

// 	// delete the new TD
// 	err = msgClient.DeleteThing(thing1ID)
// 	require.NoError(t, err)

// 	// read should fail
// 	_, err = msgClient.RetrieveThing(thing1ID)
// 	require.Error(t, err)
// }

// func TestCustomModuleIDInConfig(t *testing.T) {

// }

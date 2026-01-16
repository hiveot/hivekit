package directory_test

import (
	"testing"

	"github.com/hiveot/hivekit/go/modules/directory"
	directoryclient "github.com/hiveot/hivekit/go/modules/directory/client"
	"github.com/hiveot/hivekit/go/modules/directory/module"
	"github.com/hiveot/hivekit/go/modules/transports/direct"
	"github.com/hiveot/hivekit/go/wot/td"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var storageRoot = ""

// Generic directory store testcases
func TestStartStop(t *testing.T) {
	t.Logf("---%s---\n", t.Name())

	m := module.NewDirectoryModule(storageRoot, nil)
	err := m.Start()
	require.NoError(t, err)
	defer m.Stop()

	// add a thing
	tdJson := module.DirectoryTMJson
	m.UpdateThing(string(tdJson))

	// read all things
	tdList, err := m.RetrieveAllThings(0, 10)
	assert.NoError(t, err)
	assert.NotNil(t, tdList)
}

func TestCreateTD(t *testing.T) {
	thingID := "thing1"

	m := module.NewDirectoryModule(storageRoot, nil)
	err := m.Start()
	require.NoError(t, err)
	defer m.Stop()

	// add the directory itself
	tdJson := module.DirectoryTMJson
	m.UpdateThing(string(tdJson))

	// read all things, expect 1
	tdList, err := m.RetrieveAllThings(0, 10)
	assert.NoError(t, err)
	assert.Len(t, tdList, 1)

	// add another TD
	tdi1 := td.NewTD(thingID, "test thing", "test device")
	td1Json := tdi1.ToString()
	m.CreateThing(td1Json)

	// retrieve a thing by ID
	td2Json, err := m.RetrieveThing(thingID)
	require.NoError(t, err)
	tdi2, err := td.UnmarshalTD(td2Json)
	assert.NoError(t, err)
	assert.Equal(t, thingID, tdi2.ID)
	assert.Equal(t, td1Json, td2Json)

	// delete a thing
	err = m.DeleteThing(thingID)
	assert.NoError(t, err)

	m.Stop()
}

func TestCRUDUsingMsgAPI(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	const clientID = "user1"

	directoryID := directory.DefaultDirectoryThingID
	thing1ID := "thing1"

	m := module.NewDirectoryModule(storageRoot, nil)
	err := m.Start()
	require.NoError(t, err)
	defer m.Stop()

	// test create a TD
	tdi1 := td.NewTD(thing1ID, "thing 1", "device")
	tdi1Json := tdi1.ToString()

	// use a direct transport to the directory as the sink for the client
	tp := direct.NewDirectTransport(clientID, nil, m)
	dirClient := directoryclient.NewDirectoryMsgClient(directoryID, tp)
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

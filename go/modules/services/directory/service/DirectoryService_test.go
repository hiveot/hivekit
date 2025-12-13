package service_test

import (
	"testing"

	"github.com/hiveot/hivekit/go/modules/services/bucketstore"
	"github.com/hiveot/hivekit/go/modules/services/bucketstore/kvbtree"
	"github.com/hiveot/hivekit/go/modules/services/directory/api"
	"github.com/hiveot/hivekit/go/modules/services/directory/service"
	"github.com/hiveot/hivekit/go/wot/td"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//

// Directory service store testcases
func TestStartStop(t *testing.T) {
	var bucketStore bucketstore.IBucketStore = kvbtree.NewKVStore("") // in-memory only
	err := bucketStore.Open()
	assert.NoError(t, err)

	svc, err := service.StartDirectoryService(bucketStore, "")
	assert.NoError(t, err)
	require.NotNil(t, svc)

	err = svc.Stop()
	assert.NoError(t, err)
}

func TestCreateTD(t *testing.T) {
	thingID := "thing1"

	var bucketStore bucketstore.IBucketStore = kvbtree.NewKVStore("") // in-memory only
	err := bucketStore.Open()
	assert.NoError(t, err)

	svc, err := service.StartDirectoryService(bucketStore, "")
	require.NoError(t, err)
	require.NotNil(t, svc)
	defer svc.Stop()

	// add the directory itself
	tdJson := api.DirectoryTMJson
	svc.UpdateThing(string(tdJson))

	// read all things, expect 1
	tdList, err := svc.RetrieveAllThings(0, 10)
	assert.NoError(t, err)
	assert.Len(t, tdList, 1)

	// add another TD
	tdi1 := td.NewTD(thingID, "test thing", "test device")
	td1Json := tdi1.ToString()
	svc.CreateThing(td1Json)

	// retrieve a thing by ID
	td2Json, err := svc.RetrieveThing(thingID)
	require.NoError(t, err)
	tdi2, err := td.UnmarshalTD(td2Json)
	assert.NoError(t, err)
	assert.Equal(t, thingID, tdi2.ID)
	assert.Equal(t, td1Json, td2Json)

	// delete a thing
	err = svc.DeleteThing(thingID)
	assert.NoError(t, err)

	svc.Stop()
}

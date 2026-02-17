package history_test

import (
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/araddon/dateparse"
	"github.com/hiveot/hivekit/go/modules/bucketstore"
	"github.com/hiveot/hivekit/go/modules/clients"
	"github.com/hiveot/hivekit/go/modules/history/historyclient"
	"github.com/hiveot/hivekit/go/modules/history/module"
	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/modules/transports/tptests"
	"github.com/hiveot/hivekit/go/msg"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hivekit/go/vocab"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const thingIDPrefix = "things-"

const defaultProtocol = transports.ProtocolTypeWotWSS

// recommended store for history is Pebble
// const historyStoreBackend = bucketstore.BackendPebble
const historyStoreBackend = bucketstore.BackendKVBTree

const testClientID = "operator1"

// the following are set by the testmain
var testEnv *tptests.TestEnv

var names = []string{"temperature", "humidity", "pressure", "wind", "speed", "switch", "location", "sensor-A", "sensor-B", "sensor-C"}

func TestMain(m *testing.M) {
	var cancelFn func()
	utils.SetLogging("info", "")
	testEnv, cancelFn = tptests.StartTestEnv(defaultProtocol)
	defer cancelFn()

	res := m.Run()
	os.Exit(res)
}

// Start a history server
// This starts the protocol server and links it to the history module as sink
// Use clean to start with an empty history.
func startHistoryService(clean bool) (
	histModule *module.HistoryModule, stopFn func()) {

	if clean {
		os.RemoveAll(testEnv.StorageRoot)
	}

	// create the history module and link it to the protocol server
	// since the history module runs on the server it doesn't need an agent
	// instance.
	histModule = module.NewHistoryModule(testEnv.StorageRoot, historyStoreBackend)
	testEnv.Server.SetRequestSink(histModule.HandleRequest)
	histModule.SetNotificationSink(testEnv.Server.HandleNotification)

	err := histModule.Start("")
	if err != nil {
		panic("Failed starting the history module: " + err.Error())
	}

	return histModule, func() {
		// stop the history module, bucketstore and agent module
		histModule.Stop()
		testEnv.Server.SetRequestSink(nil)

		// give it some time to shut down before the next test
		time.Sleep(time.Millisecond)
	}
}

//func stopStore(store client.IHistory) error {
//	return store.(*mongohs.MongoHistoryServer).Stop()
//}

// generate a random batch of property and event values for testing
// timespanSec is the range of timestamps up until now
func makeValueBatch(agentID string, nrValues, nrThings, timespanSec int) (
	batch []msg.ThingValue, highest map[string]msg.ThingValue) {

	highest = make(map[string]msg.ThingValue)
	valueBatch := make([]msg.ThingValue, 0, nrValues)
	for j := 0; j < nrValues; j++ {
		randomID := rand.Intn(nrThings)
		randomName := rand.Intn(10)
		randomValue := rand.Float64() * 100
		randomSeconds := time.Duration(rand.Intn(timespanSec)) * time.Second
		randomTime := time.Now().Add(-randomSeconds)
		//
		thingID := thingIDPrefix + strconv.Itoa(randomID)
		// dThingID := td.MakeDigiTwinThingID(agentID, thingID)

		randomMsgType := rand.Intn(2)
		affType := msg.AffordanceTypeEvent
		if randomMsgType == 1 {
			affType = msg.AffordanceTypeProperty
		}

		tv := msg.ThingValue{
			//ID:             fmt.Sprintf("%d", randomID),
			SenderID:       agentID,
			Name:           names[randomName],
			Data:           fmt.Sprintf("%2.3f", randomValue),
			ThingID:        thingID,
			Timestamp:      utils.FormatUTCMilli(randomTime),
			AffordanceType: affType,
		}

		// track the actual most recent event for the name for things 3
		if randomID == 0 {
			if _, exists := highest[tv.Name]; !exists ||
				highest[tv.Name].Timestamp < tv.Timestamp {
				highest[tv.Name] = tv
			}
		}
		valueBatch = append(valueBatch, tv)
	}
	return valueBatch, highest
}

// add some history to the store. This bypasses the check for thingID to exist.
func addBulkHistory(m *module.HistoryModule, agentID string, count int, nrThings int,
	timespanSec int) (highest map[string]msg.ThingValue) {

	var batchSize = 1000
	if batchSize > count {
		batchSize = count
	}

	evBatch, highest := makeValueBatch(agentID, count, nrThings, timespanSec)

	// use add multiple in 100's
	for i := 0; i < count/batchSize; i++ {
		// no thingID constraint allows adding events from any things
		start := batchSize * i
		end := batchSize * (i + 1)
		for j := start; j < end; j++ {
			err := m.AddValue(&evBatch[j])
			if err != nil {
				slog.Error("Problem adding events.", "err", err)
			}
		}
	}
	return highest
}

// Test creating and deleting the history database
// This requires a local unsecured MongoDB instance
func TestStartStop(t *testing.T) {
	t.Logf("---%s---\n", t.Name())

	store, stopFn := startHistoryService(true)
	defer stopFn()
	_ = store
}

func TestAddGetEvent(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	const id1 = "thing1"
	const id2 = "thing2"
	const agent1ID = "agent1"
	const thing1ID = id1
	const thing2ID = id2
	const evTemperature = "temperature"
	const evHumidity = "humidity"

	// add history with two specific things to test against
	m, stopFn := startHistoryService(true)
	// do not defer cancel as it will be closed and reopened in the test

	// create an end user client for testing
	co1, _, _ := testEnv.NewConsumerClient(testClientID, transports.ClientRoleOperator, nil)
	histCl := historyclient.NewReadHistoryClient(co1)

	fivemago := time.Now().Add(-time.Minute * 5)
	fiftyfivemago := time.Now().Add(-time.Minute * 55)
	addBulkHistory(m, agent1ID, 20, 3, 3600)

	// add thing1 temperature from 5 minutes ago
	ev1_1 := &msg.NotificationMessage{
		AffordanceType: msg.AffordanceTypeEvent,
		SenderID:       agent1ID,
		ThingID:        thing1ID,
		Name:           evTemperature,
		Data:           "12.5",
		Timestamp:      utils.FormatUTCMilli(fivemago),
	}
	err := m.StoreNotification(ev1_1)
	assert.NoError(t, err)
	// add thing1 humidity from 55 minutes ago
	ev1_2 := &msg.NotificationMessage{
		AffordanceType: msg.AffordanceTypeEvent,
		SenderID:       agent1ID,
		ThingID:        thing1ID,
		Name:           evHumidity,
		Data:           "70",
		Timestamp:      utils.FormatUTCMilli(fiftyfivemago),
	}
	err = m.StoreNotification(ev1_2)
	assert.NoError(t, err)

	// add thing2 humidity from 5 minutes ago
	ev2_1 := &msg.NotificationMessage{
		AffordanceType: msg.AffordanceTypeEvent,
		SenderID:       agent1ID,
		ThingID:        thing2ID,
		Name:           evHumidity,
		Data:           "50",
		Timestamp:      utils.FormatUTCMilli(fivemago),
	}
	err = m.StoreNotification(ev2_1)
	assert.NoError(t, err)

	// add thing2 temperature from 55 minutes ago
	ev2_2 := &msg.NotificationMessage{
		AffordanceType: msg.AffordanceTypeEvent,
		SenderID:       agent1ID,
		ThingID:        thing2ID,
		Name:           evTemperature,
		Data:           "17.5",
		Timestamp:      utils.FormatUTCMilli(fiftyfivemago),
	}
	err = m.StoreNotification(ev2_2)
	assert.NoError(t, err)

	// Test 1: get events of thing1 older than 300 minutes ago - expect 1 humidity from 55 minutes ago
	cursorKey, c1Release, err := histCl.GetCursor(thing1ID, "")
	require.NoError(t, err)

	// seek must return the things humidity added 55 minutes ago, not 5 minutes ago
	timeAfter := time.Now().Add(-time.Minute * 300)
	tv1, valid, err := histCl.Seek(cursorKey, timeAfter)
	if assert.NoError(t, err) && assert.True(t, valid) {
		assert.Equal(t, thing1ID, tv1.ThingID)
		assert.Equal(t, evHumidity, tv1.Name)
		// next finds the temperature from 5 minutes ago
		tv2, valid, err := histCl.Next(cursorKey)
		assert.NoError(t, err)
		if assert.True(t, valid) {
			assert.Equal(t, evTemperature, tv2.Name)
		}
	}

	// Test 2: get events of things 1 newer than 30 minutes ago - expect 1 temperature
	timeAfter = time.Now().Add(-time.Minute * 30)

	// do we need to get a new cursor?
	//readHistory = svc.CapReadHistory()
	tv3, valid, _ := histCl.Seek(cursorKey, timeAfter)
	if assert.True(t, valid) {
		assert.Equal(t, thing1ID, tv3.ThingID)   // must match the filtered id1
		assert.Equal(t, evTemperature, tv3.Name) // must match evTemperature from 5 minutes ago
		assert.Equal(t, utils.FormatUTCMilli(fivemago), tv3.Timestamp)
	}
	c1Release()
	// Stop the service before phase 2
	stopFn()

	// PHASE 2: after closing and reopening the svc the event should still be there
	m, stopFn = startHistoryService(false)
	defer stopFn()

	// Test 3: get first temperature of things 2 - expect 1 result
	time.Sleep(time.Second)
	cursorKey, releaseFn, err := histCl.GetCursor(thing2ID, "")
	require.NoError(t, err)
	defer releaseFn()
	tv4, valid, err := histCl.First(cursorKey)
	require.NoError(t, err)
	require.NotEmpty(t, tv4)
	require.True(t, valid)
	assert.Equal(t, evTemperature, tv4.Name)

}

func TestAddProperties(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	//const clientID = "device0"
	const thing1ID = thingIDPrefix + "0" // matches a percentage of the random things
	const agent1 = "device1"
	const temp1 = 55
	const battTemp = 50

	m, closeFn := startHistoryService(true)
	defer closeFn()

	action1 := &msg.NotificationMessage{
		AffordanceType: msg.AffordanceTypeAction,
		SenderID:       agent1,
		ThingID:        thing1ID,
		Name:           vocab.ActionSwitchOnOff,
		Data:           "on",
	}
	event1 := &msg.NotificationMessage{
		AffordanceType: msg.AffordanceTypeEvent,
		SenderID:       agent1,
		ThingID:        thing1ID,
		Name:           vocab.PropEnvTemperature,
		Data:           temp1,
	}
	badEvent1 := &msg.NotificationMessage{
		AffordanceType: msg.AffordanceTypeEvent,
		SenderID:       agent1,
		ThingID:        thing1ID,
		Name:           "", // missing name
	}
	// dThing1ID identifies the publisher so not an error
	//badEvent2 := &transports.IConsumer{
	//	SenderID:  "", // missing publisher
	//	ThingID:   dThing1ID,
	//	Name:      "name",
	//	Operation: wot.OpSubscribeEvent,
	//}
	badEvent3 := &msg.NotificationMessage{
		AffordanceType: msg.AffordanceTypeEvent,
		SenderID:       agent1,
		ThingID:        thing1ID,
		Name:           "baddate",
		Timestamp:      "-1",
	}
	badEvent4 := &msg.NotificationMessage{
		AffordanceType: msg.AffordanceTypeEvent,
		SenderID:       agent1,
		ThingID:        "", // missing ID
		Name:           "temperature",
	}

	props1 := &msg.NotificationMessage{
		AffordanceType: msg.AffordanceTypeProperty,
		SenderID:       agent1,
		ThingID:        thing1ID,
		Name:           vocab.PropDeviceBattery,
		Data:           battTemp,
	}
	props2 := &msg.NotificationMessage{
		AffordanceType: msg.AffordanceTypeProperty,
		SenderID:       agent1,
		ThingID:        thing1ID,
		Name:           vocab.PropEnvCpuload,
		Data:           30,
	}
	props3 := &msg.NotificationMessage{
		AffordanceType: msg.AffordanceTypeProperty,
		SenderID:       agent1,
		ThingID:        thing1ID,
		Name:           vocab.PropSwitchOnOff,
		Data:           "off",
	}

	// in total add 5 properties
	err := m.StoreNotification(action1)
	assert.NoError(t, err)
	err = m.StoreNotification(event1)
	assert.NoError(t, err)
	err = m.StoreNotification(props1)
	assert.NoError(t, err)
	err = m.StoreNotification(props2)
	assert.NoError(t, err)
	err = m.StoreNotification(props3)
	assert.NoError(t, err)

	// and some bad values
	err = m.StoreNotification(badEvent1)
	assert.Error(t, err)
	//err = addHist.AddMessage(badEvent2)
	//assert.Error(t, err)
	err = m.StoreNotification(badEvent3) // bad date is recovered
	assert.NoError(t, err)
	err = m.StoreNotification(badEvent4)
	assert.Error(t, err)
	err = m.StoreNotification(badEvent1)
	assert.Error(t, err)

	// create an end user client for testing
	co1, _, _ := testEnv.NewConsumerClient(testClientID, transports.ClientRoleOperator, nil)
	histCl := historyclient.NewReadHistoryClient(co1)

	cursorKey, releaseFn, err := histCl.GetCursor(thing1ID, "")
	defer releaseFn()
	require.NoError(t, err)
	tv, valid, err := histCl.First(cursorKey)
	require.True(t, valid)
	assert.NotEmpty(t, tv)
	hasProps := false
	for valid && err == nil {
		if tv.AffordanceType == msg.AffordanceTypeProperty {
			hasProps = true
			require.NotEmpty(t, tv.Name)
			require.NotEmpty(t, tv.Data)
			if tv.Name == vocab.PropDeviceBattery {
				assert.Equal(t, float64(battTemp), tv.Data)
			}
			//props := make(map[string]interface{})
			//err = utils.DecodeAsObject(msg.Data, &props)
			//require.NoError(t, err)
		} else if tv.Name == vocab.PropEnvTemperature {
			dataInt := utils.DecodeAsInt(tv.Data)
			require.Equal(t, temp1, dataInt)
		}
		tv, valid, err = histCl.Next(cursorKey)
	}
	require.NoError(t, err)
	require.True(t, hasProps)
}

func TestGetInfo(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	const agentID = "agent1"
	//const thing0ID = thingIDPrefix + "0"
	//var dThing0ID = things.MakeDigiTwinThingID(agentID, thing0ID)

	store, stopFn := startHistoryService(true)
	defer stopFn()

	addBulkHistory(store, agentID, 1000, 5, 1000)

	// TODO: add GetInfo for store statistics
	//info := store.Info()
	//t.Logf("Store ID:%s, records:%d", info.Id, info.NrRecords)

	//info := readHistory.Info(ctx)
	//assert.NotEmpty(t, info.Engine)
	//assert.NotEmpty(t, info.Id)
	//t.Logf("ID:%s records:%d", info.Id, info.NrRecords)
}

func TestPrevNext(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	const count = 1000
	// const agentID = "agent1"
	const thing0ID = thingIDPrefix + "0" // matches a percentage of the random things

	store, closeFn := startHistoryService(true)
	defer closeFn()

	// 10 sensors -> 1 sample per minute, 60 per hour -> 600
	_ = addBulkHistory(store, thing0ID, count, 1, 3600*24*30)

	// create an end user client for testing
	co1, _, _ := testEnv.NewConsumerClient(testClientID, transports.ClientRoleOperator, nil)
	histCl := historyclient.NewReadHistoryClient(co1)
	cursorKey, releaseFn, err := histCl.GetCursor(thing0ID, "")
	require.NoError(t, err)
	defer releaseFn()
	assert.NotEmpty(t, cursorKey)

	// go forward
	item0, valid, err := histCl.First(cursorKey)
	require.NoError(t, err)
	assert.True(t, valid)
	assert.NotEmpty(t, item0)
	item1, valid, err := histCl.Next(cursorKey)
	require.NoError(t, err)
	assert.True(t, valid)
	assert.NotEmpty(t, item1)
	items2to11, itemsRemaining, err := histCl.NextN(cursorKey, time.Now(), 10)
	require.NoError(t, err)
	assert.True(t, itemsRemaining)
	assert.Equal(t, 10, len(items2to11))

	// go backwards
	item10to1, itemsRemaining, err := histCl.PrevN(cursorKey, time.Time{}, 10)
	require.NoError(t, err)
	assert.True(t, valid)
	assert.Equal(t, 10, len(item10to1))

	// reached first item
	item0b, valid, err := histCl.Prev(cursorKey)
	require.NoError(t, err)
	assert.True(t, valid)
	assert.Equal(t, item0.Timestamp, item0b.Timestamp)

	// can't skip before the beginning of time
	iteminv, valid, err := histCl.Prev(cursorKey)
	require.NoError(t, err)
	_ = iteminv
	assert.False(t, valid)

	// seek to item11 should succeed
	item11 := items2to11[9]
	timeStamp, _ := dateparse.ParseAny(item11.Timestamp)
	item11b, valid, err := histCl.Seek(cursorKey, timeStamp)
	require.NoError(t, err)
	assert.True(t, valid)
	assert.Equal(t, item11.Name, item11b.Name)
}

// filter on property name
func TestPrevNextFiltered(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	const count = 1000
	const agentID = "agent1"
	const thing0ID = thingIDPrefix + "0" // matches a percentage of the random things

	svc, closeFn := startHistoryService(true)
	defer closeFn()

	// 10 sensors -> 1 sample per minute, 60 per hour -> 600
	_ = addBulkHistory(svc, agentID, count, 1, 3600*24*30)
	propName := names[2] // names was used to generate the history

	// A cursor with a filter on propName should only return results of propName
	co1, _, _ := testEnv.NewConsumerClient(testClientID, transports.ClientRoleOperator, nil)
	histCl := historyclient.NewReadHistoryClient(co1)
	defer co1.Stop()
	cursorKey, releaseFn, err := histCl.GetCursor(thing0ID, propName)
	require.NoError(t, err)
	defer releaseFn()

	item0, valid, err := histCl.First(cursorKey)
	require.NoError(t, err)
	assert.True(t, valid)
	assert.Equal(t, propName, item0.Name)

	// further steps should still only return propName
	item1, valid, err := histCl.Next(cursorKey)
	assert.True(t, valid)
	require.Nil(t, err)
	assert.Equal(t, propName, item1.Name)
	items2to11, itemsRemaining, err := histCl.NextN(cursorKey, time.Now(), 10)
	assert.True(t, itemsRemaining)
	assert.Equal(t, 10, len(items2to11))
	assert.Equal(t, propName, items2to11[9].Name)

	// go backwards
	item10to1, itemsRemaining, err := histCl.PrevN(cursorKey, time.Time{}, 10)
	assert.True(t, valid)
	assert.Equal(t, 10, len(item10to1))

	// reached first item
	item0b, valid, err := histCl.Prev(cursorKey)
	assert.True(t, valid)
	require.Nil(t, err)
	assert.Equal(t, item0.Timestamp, item0b.Timestamp)
	assert.Equal(t, propName, item0b.Name)

	// can't skip before the beginning of time
	iteminv, valid, err := histCl.Prev(cursorKey)
	_ = iteminv
	assert.False(t, valid)

	// seek to item11 should succeed
	item11 := items2to11[9]
	timeStamp, _ := dateparse.ParseAny(item11.Timestamp)
	item11b, valid, err := histCl.Seek(cursorKey, timeStamp)
	assert.True(t, valid)
	require.Nil(t, err)
	assert.Equal(t, item11.Name, item11b.Name)

	// last item should be of the name
	lastItem, valid, err := histCl.Last(cursorKey)
	assert.True(t, valid)
	require.Nil(t, err)
	assert.Equal(t, propName, lastItem.Name)

}

func TestNextPrevUntil(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	const count = 1000
	const agentID = "agent1"
	const thing0ID = thingIDPrefix + "0" // matches a percentage of the random things

	store, closeFn := startHistoryService(true)
	defer closeFn()

	// 1 sensor -> 1000/24 hours is approx 41/hour
	_ = addBulkHistory(store, agentID, count, 1, 3600*24)

	co1, _, _ := testEnv.NewConsumerClient(testClientID, transports.ClientRoleOperator, nil)
	readHist := historyclient.NewReadHistoryClient(co1)
	defer co1.Stop()
	cursorKey, releaseFn, err := readHist.GetCursor(thing0ID, "")
	defer releaseFn()

	// start 20 hours ago
	startTime := time.Now().Add(-20 * time.Hour)
	item0, valid, err := readHist.Seek(cursorKey, startTime)
	require.NoError(t, err)
	assert.True(t, valid)
	assert.NotEmpty(t, item0)

	// read an hour's worth. Expect around 41 results
	endTime := startTime.Add(time.Hour)
	// note, batch1 doesn't have item0
	batch, itemsRemaining, err := readHist.NextN(cursorKey, endTime, 100)
	require.NoError(t, err)
	assert.False(t, itemsRemaining)
	assert.True(t, len(batch) > 20)
	assert.True(t, len(batch) < 60)

	// read backwards again. note batch2 ends with item0
	batch2, itemsRemaining, err := readHist.PrevN(cursorKey, startTime, 100)
	require.NoError(t, err)
	assert.False(t, itemsRemaining)
	assert.Equal(t, len(batch), len(batch2))
}

func TestReadHistory(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	const count = 1000
	const agentID = "device1"
	const thing0ID = thingIDPrefix + "0" // matches a percentage of the random things

	store, closeFn := startHistoryService(true)
	defer closeFn()
	//
	co1, _, _ := testEnv.NewConsumerClient(testClientID, transports.ClientRoleOperator, nil)
	readHist := historyclient.NewReadHistoryClient(co1)
	defer co1.Stop()

	// 1 sensors -> 1000/24 hours is approx 41/hour
	_ = addBulkHistory(store, agentID, count, 1, 3600*24)

	// start 20 hours ago and read an hour's worth
	startTime := time.Now().Add(-20 * time.Hour)
	duration := time.Hour
	items, remaining, err := readHist.ReadHistory(thing0ID, "", startTime, duration, 60)
	require.NoError(t, err)
	assert.False(t, remaining)
	assert.NotEmpty(t, items)
	assert.True(t, len(items) > 20)

	// start 19 hours ago and read back in time
	startTime = time.Now().Add(-19 * time.Hour)
	duration = -1 * time.Hour
	items, remaining, err = readHist.ReadHistory(thing0ID, "", startTime, duration, 60)
	require.NoError(t, err)
	assert.False(t, remaining)
	assert.NotEmpty(t, items)
	assert.True(t, len(items) > 20)

}

func TestPubEvents(t *testing.T) {
	const agent1ID = "device1"
	const thing1ID = "thing1"

	t.Logf("---%s---\n", t.Name())

	m, stopFn := startHistoryService(true)
	_ = m
	defer stopFn()
	co1, _, _ := testEnv.NewConsumerClient(testClientID, transports.ClientRoleOperator, nil)
	readHist := historyclient.NewReadHistoryClient(co1)
	defer co1.Stop()

	// Add the thing who is publishing events
	// td1 := testEnv.CreateTestTD(0)
	// err := m.AddValue(agent1ID, td1)
	// thing0ID := td1.ID
	// dThing0ID := td.MakeDigiTwinThingID(agent1ID, thing0ID)

	// publish events
	names := []string{
		vocab.PropEnvTemperature, vocab.PropSwitchOnOff,
		vocab.PropSwitchOnOff, vocab.PropDeviceBattery,
		vocab.PropAlarmStatus, "noname",
		"tttt", vocab.PropEnvTemperature,
		vocab.PropSwitchOnOff, vocab.PropEnvTemperature}
	_ = names

	// attach another agent after the history service so its events are recorded
	ag1 := clients.NewAgent(agent1ID, nil)
	m.SetRequestSink(ag1.HandleRequest)
	ag1.SetNotificationSink(m.HandleNotification)
	defer ag1.Start("")
	defer ag1.Stop()

	// only valid names should be added
	for i := 0; i < 10; i++ {
		val := strconv.Itoa(i + 1)
		// events are published by the agent using their native thingID
		name := names[i]
		ag1.PubEvent(thing1ID, name, val)
		// make sure timestamp differs
		time.Sleep(time.Millisecond * 3)
	}

	time.Sleep(time.Millisecond * 100)
	// read back
	// consumers read events see the digital twin representation
	cursorKey, releaseFn, err := readHist.GetCursor(thing1ID, "")
	require.NoError(t, err)
	ev, valid, err := readHist.First(cursorKey)
	require.NoError(t, err)
	assert.True(t, valid)
	assert.NotEmpty(t, ev)

	// store

	batched, _, _ := readHist.NextN(cursorKey, time.Now(), 10)
	// expect 3 entries total from valid events (9 when retention manager isn't used)
	assert.Equal(t, 9, len(batched))
	releaseFn()
}

// FIXME: this test case needs to be reworked to the new retention handling
// the storage
// func TestManageRetention(t *testing.T) {
// 	t.Logf("---%s---\n", t.Name())
// 	const client1ID = "admin"
// 	const agentID = "agent1" // should not match existing test devices
// 	const event1Name = "event1"
// 	const event2Name = "notRetainedEvent"
// 	const thingID1 = "thing-1"

// 	// setup with some history
// 	m, closeFn := startHistoryService(true)
// 	defer closeFn()
// 	addBulkHistory(m, agentID, 1000, 5, 1000)

// 	// make sure the TD whose retention rules are added exist
// 	// TODO: currently not needed.
// 	// td0 := testEnv.CreateTestTD(0)
// 	// ts.AddTD(agentID, td0)
// 	// dThing0ID := td.MakeDigiTwinThingID(agentID, td0.ID)

// 	// connect as an admin user
// 	co1, _, _ := testEnv.NewConsumerClient(client1ID, transports.ClientRoleAdmin, nil)
// 	readHist := historyclient.NewReadHistoryClient(co1)
// 	defer co1.Stop()
// 	mngHist := historyclient.NewManageHistoryClient(co1)

// 	// should be able to read the current retention rules. Expect the default rules.
// 	rules1, err := mngHist.GetRetentionRules()
// 	require.NoError(t, err)
// 	assert.Greater(t, 1, len(rules1))

// 	// Add two retention rules to retain temperature and our test event from device1
// 	rules1[vocab.PropEnvTemperature] = append(rules1[vocab.PropEnvTemperature],
// 		&historyapi.RetentionRule{Retain: true})
// 	rules1[event1Name] = append(rules1[event1Name],
// 		&historyapi.RetentionRule{Retain: true})
// 	err = mngHist.SetRetentionRules(rules1)
// 	require.NoError(t, err)

// 	// The new retention rule should now exist and accept our custom event
// 	rules2, err := mngHist.GetRetentionRules()
// 	require.NoError(t, err)
// 	assert.Equal(t, len(rules1), len(rules2))
// 	rule, err := mngHist.GetRetentionRule(thingID1, event1Name)
// 	assert.NoError(t, err)
// 	if assert.NotNil(t, rule) {
// 		assert.Equal(t, "", rule.ThingID)
// 		assert.Equal(t, event1Name, rule.Name)
// 	}

// 	// connect as agent-1 and publish two events for thing0, one to be retained
// 	ag1 := testEnv.NewServerAgent(agentID)
// 	require.NoError(t, err)
// 	defer ag1.Stop()
// 	ag1.PubEvent(thingID1, event1Name, "event one")
// 	ag1.PubEvent(thingID1, event2Name, "event two")

// 	// give it some time to persist the bucket
// 	time.Sleep(time.Millisecond * 100)

// 	// read the history of device 1 and expect the event to be retained
// 	cursorKey, releaseFn, err := readHist.GetCursor(thingID1, "")
// 	require.NoError(t, err)
// 	histEv1, valid, _ := readHist.First(cursorKey)
// 	require.True(t, valid, "missing the first event")
// 	assert.Equal(t, event1Name, histEv1.Name)
// 	histEv2, valid2, _ := readHist.Next(cursorKey)
// 	assert.False(t, valid2, "second event should not be there")
// 	_ = histEv2
// 	releaseFn()

// }

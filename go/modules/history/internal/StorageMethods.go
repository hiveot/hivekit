// Package historyserver with methods for storage, iteration and querying of thing values
package internal

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/araddon/dateparse"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/modules/bucketstore"
	"github.com/hiveot/hivekit/go/modules/history"
	"github.com/hiveot/hivekit/go/utils"
	jsoniter "github.com/json-iterator/go"
)

// decodeValue convert the storage key and raw data to a things value object
// this must match the encoding done in AddHistory
//
// If this returns an error with valid true, then the caller should ignore
// this entry and continue with the next value (if any).
//
//	bucketID is the ID of the bucket, which is the digital twin thingID
//	storageKey is the value's key, which is defined as timestamp/valueKey
//	raw is the serialized message data
//
// This returns the value, or nil if the key is invalid
// If the json in the store is invalid this returns an error
func decodeValue(bucketID string, storageKey string, raw []byte) (
	value *msg.NotificationMessage, valid bool, err error) {

	var senderID string
	// key is constructed as  timestamp/name/{a|e|c}/sender, where sender can be omitted
	parts := strings.Split(storageKey, "/")
	if len(parts) < 2 {
		// the key is invalid so return no-more-data
		return value, false, nil
	}
	createdMsec, _ := strconv.ParseInt(parts[0], 10, 64)
	createdTime := time.UnixMilli(createdMsec)
	name := parts[1]
	affType := msg.AffordanceTypeEvent
	if len(parts) >= 2 {
		if parts[2] == "a" {
			affType = msg.AffordanceTypeAction
		} else if parts[2] == "p" {
			affType = msg.AffordanceTypeProperty
		}
	}
	if len(parts) > 3 {
		senderID = parts[3]
	}
	// FIXME: keep the correlationID? serialize the ResponseMessage
	var data interface{}
	err = jsoniter.Unmarshal(raw, &data)
	if err != nil {
		// the stored data cannot be unmarshalled. This is unexpected!
		// the caller should continue with the next record as the rest of the
		// history might still be valid.
		slog.Error("decodeValue, stored data cannot be unmarshalled",
			"thingID", bucketID, "name", name, "err", err.Error())
	}

	value = msg.NewNotificationMessage(senderID, affType, bucketID, name, data)
	value.Timestamp = utils.FormatUTCMilli(createdTime)
	_ = senderID
	return value, true, err
}

// encodeValue a ResponseMessage into a single storage key value pair for easy storage and filtering.
// Encoding generates a key as: timestampMsec/name/a|e|p/sender,
// where a|e|p indicates message type "action", "event" or "property"
func encodeValue(value *msg.NotificationMessage) (storageKey string, data []byte) {
	var err error
	createdTime := time.Now().UTC()
	if value.Timestamp != "" {
		createdTime, err = dateparse.ParseAny(value.Timestamp)
		if err != nil {
			slog.Warn("Invalid Timestamp time. Using current time instead", "created", value.Timestamp)
			createdTime = time.Now().UTC()
		}
	}

	// the index uses milliseconds for timestamp
	timestamp := createdTime.UnixMilli()
	storageKey = strconv.FormatInt(timestamp, 10) + "/" + value.Name
	if value.AffordanceType == msg.AffordanceTypeAction {
		// TODO: actions subscriptions are currently not supported. This would be useful though.
		storageKey = storageKey + "/a"
	} else if value.AffordanceType == msg.AffordanceTypeProperty {
		storageKey = storageKey + "/p"
	} else { // treat everything else as events
		storageKey = storageKey + "/e"
	}
	storageKey = storageKey + "/" + value.SenderID
	//if msg.Data != nil {
	data, _ = jsoniter.Marshal(value.Data)
	//}
	return storageKey, data
}

// validateValue checks the event has the right things address, adds a timestamp if missing
// and returns if it is retained.
//
// an error will be returned if the senderID, thingID or name are empty.
func validateValue(value *msg.NotificationMessage) (err error) {
	if value.ThingID == "" {
		return fmt.Errorf("validateValue: missing thingID in value with value name '%s'", value.Name)
	}
	if value.Name == "" {
		return fmt.Errorf("validateValue: missing name for event or action for things '%s'", value.ThingID)
	}
	if value.SenderID == "" {
		return fmt.Errorf("validateValue: missing sender for action on thing '%s'", value.ThingID)
	}
	if value.Timestamp == "" {
		value.Timestamp = utils.FormatNowUTCMilli()
	}
	if value.AffordanceType != msg.AffordanceTypeProperty &&
		value.AffordanceType != msg.AffordanceTypeEvent &&
		value.AffordanceType != msg.AffordanceTypeAction {
		return fmt.Errorf("ValidateValue: Unknown affordancetype '%s'.", value.AffordanceType)
	}

	return nil
}

// AddValue adds a Thing value from a sender to the action history
// The caller must validate the SenderID in the tv.
func (svc *HistoryService) AddValue(value *msg.NotificationMessage) error {
	//slog.Info("AddValue",
	//	slog.String("senderID", senderID),
	//	slog.String("ID", tv.ID),
	//	slog.String("thingID", tv.ThingID),
	//	slog.String("name", tv.Name),
	//	slog.String("affordance", tv.AffordanceType),
	//)
	err := validateValue(value)
	if err != nil {
		slog.Info("AddValue value error", "err", err.Error())
		return err
	}
	storageKey, data := encodeValue(value)
	bucket := svc.bucketStore.GetBucket(value.ThingID)
	err = bucket.Set(storageKey, data)
	_ = bucket.Close()
	//if svc.onAddedValue != nil {
	//	svc.onAddedValue(actionValue)
	//}
	return err
}

// CreateCursor returns a new iterator for history ThingMessage objects.
//
// Unused cursors have a limited lifespan after which they are discarded.
// The default of the store is set to 1 minute. It is better to release the cursor after use.
//
//	clientID is the owner that must match iteration requests
//	thingID is the Thing whose data to iteration
//	affName of affordance to filter on or "" for any value from the thing
func (svc *HistoryService) CreateCursor(clientID string, thingID string, affName string) (cursorKey string, err error) {

	lifespan := svc.cursorLifespan

	if thingID == "" {
		return "", fmt.Errorf("missing thingID")
	}

	slog.Info("GetCursor for bucket: ", "addr", thingID)
	bucket := svc.bucketStore.GetBucket(thingID)
	cursor, err := bucket.Cursor()
	//
	if err != nil {
		return "", err
	}
	cursorKey = svc.cursorCache.Add(clientID, cursor, bucket, affName, lifespan)
	return cursorKey, nil
}

// First returns the oldest value in the history
//
// If an affordance name is provided it forwards to the first value for that affordance.
//
//	clientID must match the owner of the cursor
//	cursorKey is the cursor to iterate.
//	affName is the optional affordance name to search for, or "" for any
func (svc *HistoryService) First(clientID string, cursorKey string) (
	value *msg.NotificationMessage, valid bool, err error) {

	until := time.Now().UTC()

	cursor, ci, err := svc.cursorCache.Get(clientID, cursorKey, true)

	if err != nil {
		// invalid cursor or not the owner
		return nil, false, err
	}
	affName := ci.FilterData
	k, raw, valid := cursor.First()
	if !valid {
		// bucket is empty
		return nil, false, nil
	}

	value, valid, err = decodeValue(cursor.BucketID(), k, raw)
	// if an filter on affordance name was requested then iterate until found
	if valid && affName != "" && value.Name != affName {
		value, valid = svc.next(cursor, affName, until)
	}
	return value, valid, err
}

// Last positions the cursor at the last key in the ordered list
// If an affordance name is provided then it rewinds to the first available value
// for that affordance.
func (svc *HistoryService) Last(clientID string, cursorKey string) (
	value *msg.NotificationMessage, valid bool, err error) {

	// the beginning of time?
	until := time.Time{}
	cursor, ci, err := svc.cursorCache.Get(clientID, cursorKey, true)
	if err != nil {
		return nil, false, err
	}
	affName := ci.FilterData
	k, raw, valid := cursor.Last()

	if !valid {
		// bucket is empty
		return nil, valid, nil
	}
	value, valid, err = decodeValue(cursor.BucketID(), k, raw)

	// search back to the last valid value without an error
	if (valid || err != nil) && affName != "" && value.Name != affName {
		value, valid = svc.prev(cursor, affName, until)
	}
	return value, valid, nil
}

// next iterates the cursor until the next value containing 'name' is found and the
// timestamp doesn't exceed untilTime.
// A successive call with an increased timestamp should return the next batch of results. Intended
// to iterated an hours/day/week at a time.
// This returns the next value, or nil if the value was not found.
//
//	cursor to iterate
//	affName is the affordance name to match
//	until is the time not to exceed in the result. Intended to avoid unnecessary iteration in range queries
func (svc *HistoryService) next(
	cursor bucketstore.IBucketCursor, affName string, until time.Time) (
	value *msg.NotificationMessage, found bool) {

	untilMilli := until.UnixMilli()
	found = false
	for {
		k, raw, valid := cursor.Next()
		if !valid {
			// key is invalid. This means we reached the end of cursor
			return nil, false
		}
		// key is constructed as  {timestamp}/{affName}/{a|e|c}/{sender}
		parts := strings.Split(k, "/")
		if len(parts) != 4 {
			// key exists but is invalid. skip this entry
			slog.Warn("next: invalid key", "key", k)
		} else {
			// check timestamp and name must match
			timestampmsec, _ := strconv.ParseInt(parts[0], 10, 64)
			if untilMilli > 0 && timestampmsec > untilMilli {
				// we passed the given time limit
				// undo the last step so that followup requests with a new time limit can include this result
				cursor.Prev()
				return value, false
			}
			if affName == "" || affName == parts[1] {
				// found a match. Decode and return it
				value, found, err := decodeValue(cursor.BucketID(), k, raw)
				if err == nil {
					return value, found
				}
				// the data was invalid. ignore this entry
			}
			// name doesn't match. Skip this entry
		}
	}
}

// Read the next number of items until time or count limit is reached
func (svc *HistoryService) nextN(
	cursor bucketstore.IBucketCursor, affName string, endTime time.Time, limit int) (
	items []*msg.NotificationMessage, itemsRemaining bool) {

	items = make([]*msg.NotificationMessage, 0, limit)
	itemsRemaining = true

	for i := 0; i < limit; i++ {
		value, valid := svc.next(cursor, affName, endTime)
		if !valid {
			itemsRemaining = false
			break
		}
		items = append(items, value)
	}
	return items, itemsRemaining
}

// Next moves the cursor to the next key from the current cursor.
// If affName is provided then continue iterating until the affordance name matches.
// First() or Seek must have been called first.
// This returns an error if the cursor is not found.
func (svc *HistoryService) Next(clientID string, cursorKey string) (
	value *msg.NotificationMessage, valid bool, err error) {

	cursor, ci, err := svc.cursorCache.Get(clientID, cursorKey, true)
	if err != nil {
		return nil, false, err
	}
	affName := ci.FilterData
	until := time.Now()
	value, valid = svc.next(cursor, affName, until)

	return value, valid, nil
}

// NextN moves the cursor to the next N places from the current cursor and return a
// list with N values in incremental time order.
//
// This returns the list with values and itemsRemaining, which is false if the iterator
// has reached the end.
// Intended to speed up with batch iterations over rpc.
func (svc *HistoryService) NextN(
	clientID string, cursorKey string, until time.Time, limit int) (
	valueList []*msg.NotificationMessage, itemsRemaining bool, err error) {

	if limit <= 0 {
		limit = history.DefaultLimit
	}
	cursor, ci, err := svc.cursorCache.Get(clientID, cursorKey, true)
	if err != nil {
		return nil, false, err
	}
	affName := ci.FilterData
	valueList, itemsRemaining = svc.nextN(cursor, affName, until, limit)
	return valueList, itemsRemaining, err
}

// Prev iterates the cursor until the previous value passes the filters and the
// timestamp is not before 'until' time.
//
// This supports 2 filters, a key of the value and a timestamp.
// Since key and timestamp are part of the bucket key these checks are fast.
//
// This returns the previous value, or nil if the value was not found.
//
//	cursor is a valid bucket cursor
//	affName is the value affordance name (event,prop,action name) to match or "" for any.
//	until is the limit of the time to read. Intended for time-range queries and
//	to avoid unnecessary iteration in range queries
func (svc *HistoryService) prev(
	cursor bucketstore.IBucketCursor, affName string, until time.Time) (
	value *msg.NotificationMessage, found bool) {

	untilMilli := until.UnixMilli()
	found = false
	for {
		k, raw, valid := cursor.Prev()
		if !valid {
			// key is invalid. This means we reached the beginning of cursor
			return value, false
		}
		// key is constructed as  {timestamp}/{affName}/{a|e|c}/sender
		parts := strings.Split(k, "/")
		if len(parts) != 4 {
			// key exists but is invalid. skip this entry
			slog.Warn("prev: invalid key", "key", k)
		} else {
			// check timestamp and name must match
			timestampmsec, _ := strconv.ParseInt(parts[0], 10, 64)
			if timestampmsec < untilMilli {
				// we passed the given time limit
				// undo the last step so that followup requests with a new time limit can include this result
				cursor.Next()
				return nil, false
			}

			if affName == "" || affName == parts[1] {
				// found a match. Decode and return it
				value, found, err := decodeValue(cursor.BucketID(), k, raw)
				if err == nil {
					return value, found
				}
				// the data was invalid for unknown reason. Skip this entry.
			}
			// filter doesn't match. Skip this entry
		}
	}
}

// prevN reads the previous number of items until time or count limit is reached
func (svc *HistoryService) prevN(
	cursor bucketstore.IBucketCursor, affName string, endTime time.Time, limit int) (
	items []*msg.NotificationMessage, itemsRemaining bool) {

	items = make([]*msg.NotificationMessage, 0, limit)
	itemsRemaining = true

	for i := 0; i < limit; i++ {
		value, valid := svc.prev(cursor, affName, endTime)
		if !valid {
			itemsRemaining = false
			break
		}
		items = append(items, value)
	}
	return items, itemsRemaining
}

// Prev moves the cursor to the previous key from the current cursor
// Last() or Seek must have been called first.
// This returns an error if the cursor is not found.
func (svc *HistoryService) Prev(clientID string, cursorKey string) (
	value *msg.NotificationMessage, valid bool, err error) {

	cursor, ci, err := svc.cursorCache.Get(clientID, cursorKey, true)
	if err != nil {
		return nil, false, err
	}
	affName := ci.FilterData
	until := time.Time{}
	value, valid = svc.prev(cursor, affName, until)

	return value, valid, nil
}

// PrevN moves the cursor to the previous N places from the current cursor
// and return a list with N values in reverse time order.
//
// itemsRemaining returns false if the iterator has reached the beginning.
// Intended to speed up with batch iterations over rpc.
func (svc *HistoryService) PrevN(
	clientID string, cursorKey string, until time.Time, limit int) (
	valueList []*msg.NotificationMessage, itemsRemaining bool, err error) {

	if limit <= 0 {
		limit = history.DefaultLimit
	}
	cursor, ci, err := svc.cursorCache.Get(clientID, cursorKey, true)
	if err != nil {
		return nil, false, err
	}
	affName := ci.FilterData
	valueList, itemsRemaining = svc.prevN(cursor, affName, until, limit)
	return valueList, itemsRemaining, err
}

// ReadHistory returns the history for the given thingID, name and time range
func (svc *HistoryService) ReadHistory(
	thingID string, affName string, timestamp time.Time, durationSec int, limit int) (
	values []*msg.NotificationMessage, itemsRemaining bool, err error) {

	values = make([]*msg.NotificationMessage, 0)

	if limit <= 0 {
		limit = history.DefaultLimit
	}
	if thingID == "" {
		return nil, false, fmt.Errorf("missing thingID")
	}
	bucket := svc.bucketStore.GetBucket(thingID)
	cursor, err := bucket.Cursor()
	if err != nil {
		return nil, false, err
	}
	defer cursor.Release()

	item0, valid := svc.seek(cursor, timestamp, affName)
	if valid {
		// item0 is nil when seek after the last available item
		values = append(values, item0)
	}
	var batch []*msg.NotificationMessage
	until := timestamp.Add(time.Duration(durationSec) * time.Second)
	if durationSec > 0 {
		// read forward in time
		batch, itemsRemaining = svc.nextN(cursor, affName, until, limit)
	} else {
		// read backwards in time
		batch, itemsRemaining = svc.prevN(cursor, affName, until, limit)
	}
	values = append(values, batch...)
	return values, itemsRemaining, err
}

// ReleaseCursor frees the cursor resources.
// This invalidates all values obtained from the cursor
func (svc *HistoryService) ReleaseCursor(clientID string, cursorKey string) error {

	return svc.cursorCache.Release(clientID, cursorKey)
}

// seek internal function for seeking a time and affordance name
func (svc *HistoryService) seek(
	cursor bucketstore.IBucketCursor, ts time.Time, affName string) (
	value *msg.NotificationMessage, valid bool) {

	until := time.Now()

	// search the first occurrence at or after the given timestamp
	// the bucket index uses the stringified timestamp
	msec := ts.UnixMilli()
	searchKey := strconv.FormatInt(msec, 10)

	k, raw, valid := cursor.Seek(searchKey)
	if !valid {
		// bucket is empty, no error
		return nil, valid
	}
	value, valid, err := decodeValue(cursor.BucketID(), k, raw)
	if err != nil {
		// the value cannot be decoded, skip this entry
		value, valid = svc.next(cursor, affName, until)
	} else if valid && affName != "" && value.Name != affName {
		value, valid = svc.next(cursor, affName, until)
	}
	return value, valid
}

// Seek positions the cursor at the given time stamp and affordance name.
// If the key is not found, the next key is returned.
// This returns an error if the cursor is not valid.
func (svc *HistoryService) Seek(
	clientID string, cursorKey string, ts time.Time) (
	value *msg.NotificationMessage, valid bool, err error) {

	slog.Info("Seek using timestamp",
		slog.Time("timestamp", ts),
	)

	cursor, ci, err := svc.cursorCache.Get(clientID, cursorKey, true)
	if err != nil {
		return nil, false, err
	}
	affName := ci.FilterData

	// search the first occurrence at or after the given timestamp
	// the buck index uses the stringified timestamp
	value, valid = svc.seek(cursor, ts, affName)

	return value, valid, err
}

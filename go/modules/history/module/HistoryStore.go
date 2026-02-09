package module

import (
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/araddon/dateparse"
	"github.com/hiveot/hivekit/go/modules/bucketstore"
	"github.com/hiveot/hivekit/go/msg"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hivekit/go/wot"
	jsoniter "github.com/json-iterator/go"
)

// HistoryStore provides the underlying storage for notifications and requests.
type HistoryStore struct {
	// store with a bucket for each Thing
	store bucketstore.IBucketStore
}

// AddMessage adds the value of an event or property notification to the history store.
func (svc *HistoryStore) AddNotification(notif *msg.NotificationMessage) error {

	// convert the notification to a ThingValue for storage
	tv := msg.NewThingValue(
		notif.SenderID,
		notif.AffordanceType,
		notif.ThingID,
		notif.Name,
		notif.Data,
		notif.Timestamp,
	)
	err := svc.AddValue(tv)
	return err
}

// AddAction adds the action request to the history store.
func (svc *HistoryStore) AddAction(req *msg.RequestMessage) error {

	if req.Operation != wot.OpInvokeAction {
		return fmt.Errorf("AddAction: Operation is not invokeaction")
	}
	// convert the notification to a ThingValue for storage
	tv := msg.NewThingValue(
		req.SenderID,
		msg.AffordanceTypeAction,
		req.ThingID,
		req.Name,
		req.Input,
		req.Created,
	)
	err := svc.AddValue(tv)
	return err
}

// AddValue adds a Thing value from a sender to the action history
func (svc *HistoryStore) AddValue(tv *msg.ThingValue) error {
	//slog.Info("AddValue",
	//	slog.String("senderID", senderID),
	//	slog.String("ID", tv.ID),
	//	slog.String("thingID", tv.ThingID),
	//	slog.String("name", tv.Name),
	//	slog.String("affordance", tv.AffordanceType),
	//)
	err := svc.validateValue(tv)
	if err != nil {
		slog.Info("AddValue value error", "err", err.Error())
		return err
	}
	storageKey, val := svc.encodeValue(tv)
	bucket := svc.store.GetBucket(tv.ThingID)
	err = bucket.Set(storageKey, val)
	_ = bucket.Close()
	//if svc.onAddedValue != nil {
	//	svc.onAddedValue(actionValue)
	//}
	return err
}

// encode a ResponseMessage into a single storage key value pair for easy storage and filtering.
// Encoding generates a key as: timestampMsec/name/a|e|p/sender,
// where a|e|p indicates message type "action", "event" or "property"
func (svc *HistoryStore) encodeValue(tv *msg.ThingValue) (storageKey string, data []byte) {
	var err error
	createdTime := time.Now().UTC()
	if tv.Timestamp != "" {
		createdTime, err = dateparse.ParseAny(tv.Timestamp)
		if err != nil {
			slog.Warn("Invalid Timestamp time. Using current time instead", "created", tv.Timestamp)
			createdTime = time.Now().UTC()
		}
	}

	// the index uses milliseconds for timestamp
	timestamp := createdTime.UnixMilli()
	storageKey = strconv.FormatInt(timestamp, 10) + "/" + tv.Name
	if tv.AffordanceType == msg.AffordanceTypeAction {
		// TODO: actions subscriptions are currently not supported. This would be useful though.
		storageKey = storageKey + "/a"
	} else if tv.AffordanceType == msg.AffordanceTypeProperty {
		storageKey = storageKey + "/p"
	} else { // treat everything else as events
		storageKey = storageKey + "/e"
	}
	storageKey = storageKey + "/" + tv.SenderID
	//if msg.Data != nil {
	data, _ = jsoniter.Marshal(tv.Data)
	//}
	return storageKey, data
}

// validateValue checks the event has the right things address, adds a timestamp if missing
// and returns if it is retained.
//
// an error will be returned if the senderID, thingID or name are empty.
func (svc *HistoryStore) validateValue(tv *msg.ThingValue) (err error) {
	if tv.ThingID == "" {
		return fmt.Errorf("validateValue: missing thingID in value with value name '%s'", tv.Name)
	}
	if tv.Name == "" {
		return fmt.Errorf("validateValue: missing name for event or action for things '%s'", tv.ThingID)
	}
	if tv.SenderID == "" {
		return fmt.Errorf("validateValue: missing sender for action on thing '%s'", tv.ThingID)
	}
	if tv.Timestamp == "" {
		tv.Timestamp = utils.FormatNowUTCMilli()
	}
	if tv.AffordanceType != msg.AffordanceTypeProperty &&
		tv.AffordanceType != msg.AffordanceTypeEvent &&
		tv.AffordanceType != msg.AffordanceTypeAction {
		return fmt.Errorf("validateValue: Unknown affordancetype '%s'.", tv.AffordanceType)
	}

	return nil
}

// NewHistoryStore persists and retrieves historical events, properties or actions
// into the given bucket store.
//
//	store with a bucket for each Thing
func NewHistoryStore(store bucketstore.IBucketStore) *HistoryStore {
	svc := &HistoryStore{
		store: store,
		// MaxMessageSize: DefaultMaxMessageSize,
	}

	return svc
}

package consumer_test

import (
	"log/slog"
	"testing"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/wot"
	"github.com/hiveot/hivekit/go/cells/consumer"
	"github.com/hiveot/hivekit/go/utils"

	"github.com/stretchr/testify/assert"
)

const key1ID = "key1"

var testNotif = &msg.NotificationMessage{Name: key1ID, Data: nil, AffordanceType: msg.AffordanceTypeProperty}

func MakeTD() *wot.TD {
	tdoc := wot.NewTD("thingID", "test Thing", "devicetype")
	tdoc.AddProperty(key1ID, "property 1", "test property", wot.DataTypeString)
	return tdoc
}

func TestNilSchema(t *testing.T) {
	utils.SetLogging("", "")
	slog.Info("--- TestNilSchema ---")
	data1 := "text"

	notif := *testNotif
	notif.Data = data1
	tdoc := MakeTD()
	io := consumer.NewInteractionOutputFromNotification(tdoc, &notif)

	asValue := io.Value.Text()
	assert.Equal(t, data1, asValue)

}

func TestArray(t *testing.T) {
	notif := *testNotif
	notif.Data = []string{"item 1", "item 2"}
	tdoc := MakeTD()
	io := consumer.NewInteractionOutputFromNotification(tdoc, &notif)
	asArray := io.Value.Array()
	assert.Len(t, asArray, 2)
}

func TestBool(t *testing.T) {

	notif := *testNotif
	notif.Data = true
	tdoc := MakeTD()
	io := consumer.NewInteractionOutputFromNotification(tdoc, &notif)
	asBool := io.Value.Boolean()
	assert.Equal(t, true, asBool)
	asString := io.Value.Text()
	assert.Equal(t, "true", asString)
	asInt := io.Value.Integer()
	assert.Equal(t, 1, asInt)
}

func TestInt(t *testing.T) {
	notif := *testNotif
	notif.Data = 42
	tdoc := MakeTD()
	io := consumer.NewInteractionOutputFromNotification(tdoc, &notif)
	asInt := io.Value.Integer()
	assert.Equal(t, 42, asInt)
	asString := io.Value.Text()
	assert.Equal(t, "42", asString)
}

func TestString(t *testing.T) {
	notif := *testNotif
	notif.Data = "Hello world"
	tdoc := MakeTD()
	io := consumer.NewInteractionOutputFromNotification(tdoc, &notif)
	asString := io.Value.Text()
	assert.Equal(t, notif.Data, asString)
}

func TestObject(t *testing.T) {
	//schema := tdd.DataSchema{Type: vocab.DataTypeObject}
	type User struct {
		Name        string
		Age         int
		Active      bool
		LastLoginAt string
	}
	data1 := User{Name: "Bob", Age: 10, Active: true, LastLoginAt: "today"}
	notif := *testNotif
	notif.Data = data1
	tdoc := MakeTD()
	io := consumer.NewInteractionOutputFromNotification(tdoc, &notif)
	asMap := io.Value.Map()
	assert.Equal(t, data1.Name, asMap["Name"])
}

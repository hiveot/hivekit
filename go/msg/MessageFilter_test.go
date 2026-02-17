package msg_test

import (
	"testing"

	"github.com/hiveot/hivekit/go/msg"
	"github.com/hiveot/hivekit/go/wot"
	"github.com/stretchr/testify/assert"
)

var ev1 = msg.NewNotificationMessage("sender1", msg.AffordanceTypeEvent, "thing1", "event1", nil)
var ev2 = msg.NewNotificationMessage("sender1", msg.AffordanceTypeEvent, "thing2", "event2", nil)
var ev3 = msg.NewNotificationMessage("sender1", msg.AffordanceTypeEvent, "thing2", "event3", nil)
var ev4 = msg.NewNotificationMessage("sender1", msg.AffordanceTypeEvent, "thing2", "event4", nil)
var prop1 = msg.NewNotificationMessage("sender1", msg.AffordanceTypeProperty, "thing1", "property1", nil)
var act1 = msg.NewNotificationMessage("sender1", msg.AffordanceTypeAction, "thing1", "action1", nil)

var reqProp1 = msg.NewRequestMessage(wot.OpWriteProperty, "thing1", "property1", nil, "")
var reqAct2 = msg.NewRequestMessage(wot.OpInvokeAction, "thing1", "action1", nil, "")

func TestAcceptEmpty(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	f1 := msg.MessageFilter{}
	accepted := f1.AcceptNotification(ev1)
	assert.True(t, accepted)
	accepted = f1.AcceptNotification(prop1)
	assert.True(t, accepted)
	accepted = f1.AcceptNotification(act1)
	assert.True(t, accepted)

	accepted = f1.AcceptRequest(reqProp1)
	assert.True(t, accepted)
	accepted = f1.AcceptRequest(reqAct2)
	assert.True(t, accepted)
}

func TestAcceptPass(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	f1 := msg.MessageFilter{
		Events: msg.MessageFilterChain{
			msg.MessageFilterStep{ThingID: "thing1", Names: []string{"event1"}, Accept: true},
		},
		Properties: msg.MessageFilterChain{
			msg.MessageFilterStep{ThingID: "thing1", Names: []string{"property1"}, Accept: true},
		},
		Actions: msg.MessageFilterChain{
			msg.MessageFilterStep{ThingID: "thing1", Names: []string{"action1"}, Accept: true},
		},
	}
	accepted := f1.AcceptNotification(ev1)
	assert.True(t, accepted)
	accepted = f1.AcceptNotification(prop1)
	assert.True(t, accepted)
	accepted = f1.AcceptNotification(act1)
	assert.True(t, accepted)

	accepted = f1.AcceptRequest(reqProp1)
	assert.True(t, accepted)
	accepted = f1.AcceptRequest(reqAct2)
	assert.True(t, accepted)
}

func TestAcceptReject(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	f1 := msg.MessageFilter{
		Events: msg.MessageFilterChain{
			msg.MessageFilterStep{ThingID: "thing2", Names: []string{"event1"}, Accept: true},
		},
		Properties: msg.MessageFilterChain{
			msg.MessageFilterStep{ThingID: "thing2", Names: []string{"property1"}, Accept: true},
		},
		Actions: msg.MessageFilterChain{
			msg.MessageFilterStep{ThingID: "thing2", Names: []string{"action1"}, Accept: true},
		},
	}
	accepted := f1.AcceptNotification(ev1)
	assert.False(t, accepted)
	accepted = f1.AcceptNotification(prop1)
	assert.False(t, accepted)
	accepted = f1.AcceptNotification(act1)
	assert.False(t, accepted)

	accepted = f1.AcceptRequest(reqProp1)
	assert.False(t, accepted)
	accepted = f1.AcceptRequest(reqAct2)
	assert.False(t, accepted)
}

// test filtering on multiple criteria
func TestAcceptMultistep(t *testing.T) {
	t.Logf("---%s---\n", t.Name())

	f1 := msg.MessageFilter{
		Events: msg.MessageFilterChain{
			msg.MessageFilterStep{ThingID: "thing1", Names: []string{"event1"}, Accept: true},
			msg.MessageFilterStep{ThingID: "thing2", Names: []string{"event2", "event3"}, Accept: true},
		},
	}
	accepted := f1.AcceptNotification(ev1)
	assert.True(t, accepted)
	accepted = f1.AcceptNotification(ev2)
	assert.True(t, accepted)
	accepted = f1.AcceptNotification(ev3)
	assert.True(t, accepted)
	accepted = f1.AcceptNotification(ev4)
	assert.False(t, accepted)
}

package reconnect_test

import (
	"context"
	"errors"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/modules/authn"
	"github.com/hiveot/hivekit/go/modules/consumer"
	"github.com/hiveot/hivekit/go/modules/thing"
	"github.com/hiveot/hivekit/go/testenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testProtocol = api.ProtocolTypeHiveotGrpc

var testProtocols = []string{
	api.ProtocolTypeHiveotSsesc,
	api.ProtocolTypeHiveotGrpc,
	api.ProtocolTypeHiveotWebsocket,
	api.ProtocolTypeWotWebsocket,
}

const testClientID1 = "client1"

func TestReconnectAllProtocols(t *testing.T) {
	for _, testProtocol = range testProtocols {
		t.Run(testProtocol, TestReconnect)
	}
}

// Reconnect the client using the 'ReconnectClient' module
// TODO: move this as a test case of the Reconnect module)
func TestReconnect(t *testing.T) {
	t.Logf("---%s %s---\n", t.Name(), testProtocol)

	const thingID = "thing1"
	const actionKey = "action1"
	const deviceID = "device1"
	var serverConnectEvents atomic.Int32
	var clientConnectEvents atomic.Int32

	// check if the connection status notification is received
	// The notification is experimental, reconnect uses the client callback which
	// comes after the notification.
	notificationHook := func(notif *msg.NotificationMessage) {
		if notif.Name == api.ClientConnectionStatusEvent {
			status := notif.Data.(api.ConnectionStatus)
			slog.Info("TestReconnect: client connection notification", "status", status)
			clientConnectEvents.Add(1)
		}
	}

	// this test device receives an action and returns the input
	// it is intended to prove reconnect works.
	ag := thing.NewExposedThing("", func(req *msg.RequestMessage, replyTo msg.ResponseHandler) error {
		slog.Info("Received request", "op", req.Operation)
		var err error
		// prove that the return channel is connected
		if req.Operation == td.OpInvokeAction {
			go func() {
				// send an asynchronous result after a short time
				time.Sleep(time.Millisecond * 10)
				output := req.Input

				resp := req.CreateResponse(output, nil)
				resp.SenderID = deviceID
				err = replyTo(resp)
				assert.NoError(t, err)
			}()
			// nothing to return yet
			return nil
		}
		err = errors.New("unexpected request")
		resp := req.CreateResponse("", err)
		// err = c.SendResponse(resp)
		err = replyTo(resp)
		return err
	})
	// start the servers and handle a request
	testEnv, cancelFn := testenv.StartTestEnv(testProtocol, true)
	testEnv.Server.SetRequestSink(ag)
	// server emits notification when a new connection is received
	notifHandler := consumer.NewConsumer(nil, func(notif *msg.NotificationMessage) {
		if notif.Name == api.ServerConnectEvent {
			// expect a connect-disconnect event
			serverConnectEvents.Add(1)
			slog.Info("TestReconnect: Connection notification by Server",
				slog.String("type", string(notif.AffordanceType)),
				slog.String("thingID", notif.ThingID),
				slog.String("name", notif.Name),
			)
		}
	})
	testEnv.Server.SetNotificationSink(notifHandler)
	defer cancelFn()

	// connect as consumer and give client a second to reconnect
	ctx1, cancelFn1 := context.WithTimeout(context.Background(), time.Second)
	defer cancelFn1()
	// new consumer with reconnect client
	co1, cc1, _ := testEnv.NewConnectedConsumer(
		testClientID1, authn.ClientRoleViewer, true)
	// cc1.SetConnectHandler(handleConnect)
	co1.SetNotificationHook(notificationHook)
	defer cc1.Close()

	//  wait until the connection is established

	// 3. close connection server side but keep the session.
	// This should trigger auto-reconnect on the client.
	slog.Warn("--- force disconnecting all clients. This can log a warning ---")
	testEnv.Server.CloseAll()

	<-ctx1.Done()

	// 4. invoke an action which should return a value
	// An RPC call is the ultimate test
	var rpcArgs = "rpc test"
	var rpcResp string
	time.Sleep(time.Millisecond * 1000)
	err := co1.InvokeAction(thingID, actionKey, &rpcArgs, &rpcResp)
	require.NoError(t, err)
	assert.Equal(t, rpcArgs, rpcResp)
	// expect connect and reconnect = 2
	// Note: there might be a timing issue where only 1 event is received
	assert.Equal(t, 2, int(serverConnectEvents.Load()), "duplicate or missing server connections")
	// lost, connecting, connected = 3
	assert.GreaterOrEqual(t, int(clientConnectEvents.Load()), 3, "missing client connection callbacks")
}

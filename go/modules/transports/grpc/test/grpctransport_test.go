package grpc_test

import (
	"fmt"
	"log/slog"
	"net"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hiveot/hivekit/go/modules/transports/grpc/internal/grpcclient"
	"github.com/hiveot/hivekit/go/modules/transports/grpc/internal/grpcserver"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var address = "/tmp/hivekit/grpc-test.sock" // host[:port]
var network = "unix"                        // unix, tcp, tcp4, tcp6
var scheme = "unix"                         // unix, dns, ipv4, ipv6

// TestMain runs a gRPC server
func TestMain(m *testing.M) {
	utils.SetLogging("info", "")
	// slog.Info("------ TestMain of TLSServer_test.go ------")
	// serverAddress = utils.GetOutboundIP("").String()
	// use the localhost interface for testing
	os.MkdirAll("/tmp/hivekit", 0700)
	os.Remove(address)

	res := m.Run()

	time.Sleep(time.Second)
	os.Exit(res)
}

func TestConnectPing(t *testing.T) {
	// test connect/disconnect with ping
	t.Logf("---%s---\n", t.Name())
	const clientID = "client1"
	var connectCount atomic.Int32

	// setup the server
	handleServiceMessage := func(msgType string, jsonRaw string) {
	}
	handleServiceConnection := func(strm *grpcserver.GrpcServiceStream) error {
		connectCount.Add(1)
		strm.Run(handleServiceMessage)
		return nil
	}
	lis, err := net.Listen(network, address)
	require.NoError(t, err)

	srv, err := grpcserver.StartGrpcServiceServer(lis, nil,
		handleServiceConnection, time.Minute)
	require.NoError(t, err)
	defer srv.Stop()

	// connect with the client
	handleClientMessage := func(msgType string, jsonRaw string) {
	}
	handleClientConnected := func(connected bool, err error) {
	}
	serverURL := fmt.Sprintf("%s://%s", scheme, address)
	cl := grpcclient.NewGrpcServiceClient(
		serverURL, time.Minute, handleClientMessage, handleClientConnected)

	err = cl.Connect()
	require.NoError(t, err)

	// test ping
	reply, err := cl.Ping("hello")
	assert.NoError(t, err)
	assert.Equal(t, "hello", reply)

	assert.Equal(t, 1, int(connectCount.Load()))

	cl.Close()

	// check closing twice not causing a panic
	assert.NotPanics(t, func() {
		cl.Close()
	})
}

func TestStreamToServer(t *testing.T) {
	// test streaming to server by multiple clients
	t.Logf("---%s---\n", t.Name())
	const clientID = "client1"
	var connectCount atomic.Int32
	var serviceCustomMsgType = "serviceMessagetype"
	var txClientMsg string = "client hello"
	var txServerMsg string = "server hello"
	var rxMsg string
	var svc *grpcserver.GrpcServiceServer

	// setup the server
	handleServiceMessage := func(msgType string, jsonRaw string) {
		slog.Info("test: service received message", "msgType", msgType)
		rxMsg = jsonRaw
	}
	handleServiceConnect := func(strm *grpcserver.GrpcServiceStream) error {
		go func() {
			strm.Run(handleServiceMessage)
			slog.Info("test: handleServiceMessage ended")
		}()
		connectCount.Add(1)
		go func() {
			slog.Info("test: service sending message", "msgType", serviceCustomMsgType)
			err := strm.Send(serviceCustomMsgType, txServerMsg)
			assert.NoError(t, err)
		}()
		return nil
	}
	lis, err := net.Listen(network, address)
	require.NoError(t, err)

	svc, err = grpcserver.StartGrpcServiceServer(lis, nil,
		handleServiceConnect, time.Minute)
	require.NoError(t, err)
	defer svc.Stop()

	// connect the client and receive the server message
	onClientMessage := func(msgType string, jsonRaw string) {
		slog.Info("test: Client received message", "msgType", msgType)
		rxMsg = jsonRaw
	}
	onClientConnect := func(connected bool, err error) {
		slog.Info("test: Client connection: ", "connected", connected)
	}
	serverURL := fmt.Sprintf("%s://%s", scheme, address)
	cl := grpcclient.NewGrpcServiceClient(serverURL, time.Minute,
		onClientMessage, onClientConnect)

	err = cl.Connect()
	require.NoError(t, err)
	defer cl.Close()
	cl.Run()

	cl.Send("notification", txClientMsg)
	// fixme wait for signal or timeout
	time.Sleep(time.Millisecond * 100)
	assert.Equal(t, txClientMsg, rxMsg)

	// connect should have triggered a message from the server
	time.Sleep(time.Millisecond * 100)
	assert.Equal(t, txServerMsg, rxMsg)
	t.Log("test: Shutting down")

	cl.Close()
}

func TestStreamToClient(t *testing.T) {
	// test streaming to multiple clients
}

func TestMultipleClients(t *testing.T) {
	// test multiple clients connecting and disconnecting rapidly
}

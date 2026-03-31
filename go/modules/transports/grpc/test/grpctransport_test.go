package grpc_test

import (
	"fmt"
	"log/slog"
	"net"
	"os"
	"sync/atomic"
	"testing"
	"time"

	grpcapi "github.com/hiveot/hivekit/go/modules/transports/grpc/api"
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

	// setup the server
	serveStream := func(grpcStream grpcapi.GrpcService_MsgStreamServer) error {
		return nil
	}
	lis, err := net.Listen(network, address)
	require.NoError(t, err)

	srv, err := grpcserver.StartGrpcServiceServer(lis, nil,
		serveStream, time.Minute)
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
	assert.Equal(t, "pong", reply)

	cl.Close()

	// check closing twice not causing a panic
	assert.NotPanics(t, func() {
		cl.Close()
	})
}

func TestStreamMessages(t *testing.T) {
	// test streaming to server by multiple clients
	t.Logf("---%s---\n", t.Name())
	const clientID = "client1"
	var msgCount atomic.Int32
	var serviceCustomMsgType = "serviceMessagetype"
	var clientSendMsg string = "client hello"
	var serverSendMsg string = "server hello"
	var svc *grpcserver.GrpcServiceServer

	// setup the server
	handleServiceMessage := func(msgType string, jsonRaw string) {
		slog.Info("test: service received message", "msgType", msgType)
		assert.Equal(t, clientSendMsg, jsonRaw)
		msgCount.Add(1)
	}
	serveStream := func(grpcStream grpcapi.GrpcService_MsgStreamServer) error {
		// start the send and receive loop
		bstrm := grpcclient.NewGrpcBufferedStream(grpcStream, handleServiceMessage, time.Minute)

		// send is dispatched after the stream is
		slog.Info("test: serveStream sending message", "msgType", serviceCustomMsgType)
		err := bstrm.Send(serviceCustomMsgType, serverSendMsg)
		assert.NoError(t, err)

		// start the send and receive loop
		bstrm.WaitUntilDisconnect()
		// must block until connection closes
		slog.Info("test: serveStream ended")

		return nil
	}
	lis, err := net.Listen(network, address)
	require.NoError(t, err)

	svc, err = grpcserver.StartGrpcServiceServer(lis, nil,
		serveStream, time.Minute)
	require.NoError(t, err)
	defer svc.Stop()

	time.Sleep(time.Millisecond * 100)

	// connect the client and receive the server message
	onClientMessage := func(msgType string, jsonRaw string) {
		slog.Info("test: Client received message", "msgType", msgType)
		msgCount.Add(1)
		assert.Equal(t, serverSendMsg, jsonRaw)
	}
	onClientConnect := func(connected bool, err error) {
		slog.Info("test: Client connection: ", "connected", connected)
	}
	serverURL := fmt.Sprintf("%s://%s", scheme, address)
	cl := grpcclient.NewGrpcServiceClient(serverURL, time.Minute,
		onClientMessage, onClientConnect)

	err = cl.Connect()
	require.NoError(t, err)
	// defer cl.Close()
	// run blocks until the stream is closed
	go cl.Run()

	cl.Send("notification", clientSendMsg)
	time.Sleep(time.Millisecond * 10)

	// both client and server side should have received a message
	assert.Equal(t, 2, int(msgCount.Load()))

	// graceful shutdown no errors or warndings are expected
	t.Log("test: Closing client")
	cl.Close()

	time.Sleep(time.Second * 1)
	t.Log("test: end of test - defer close of server")
}

func TestMultipleClients(t *testing.T) {
	// test multiple clients sending messages concurrently
}

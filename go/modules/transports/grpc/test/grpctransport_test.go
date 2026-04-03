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
	const clientID = "client1"
	const token = "secret1"

	// setup the server
	serveStream := func(clientID string, cid string, grpcStream grpcapi.GrpcService_MsgStreamServer) error {
		return nil
	}
	lis, err := net.Listen(network, address)
	require.NoError(t, err)

	srv, err := grpcserver.StartGrpcServiceServer(
		lis, nil, serveStream, nil, time.Minute)
	require.NoError(t, err)
	defer srv.Stop()

	// connect with the client
	handleClientMessage := func(msgType string, jsonRaw string) {
	}

	serverURL := fmt.Sprintf("%s://%s", scheme, address)
	cl := grpcclient.NewGrpcServiceClient(clientID, serverURL, nil, time.Minute, handleClientMessage)

	err = cl.ConnectWithToken(token)
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
	var authToken = "token1"
	var msgCount atomic.Int32
	var clientConnectCount atomic.Int32
	var serviceCustomMsgType = "serviceMessagetype"
	var clientSendMsg string = "client hello"
	var serverSendMsg string = "server hello"
	var svc *grpcserver.GrpcServiceServer

	// setup the server
	handleServiceMessage := func(msgType string, jsonRaw string) {
		assert.Equal(t, clientSendMsg, jsonRaw)
		msgCount.Add(1)
	}
	serveStream := func(clientID, cid string, grpcStream grpcapi.GrpcService_MsgStreamServer) error {
		// todo test authentication?

		// start the send and receive loop
		bstrm := grpcclient.NewGrpcBufferedStream(grpcStream, handleServiceMessage, time.Minute)

		// send is dispatched after the stream is
		err := bstrm.Send(serviceCustomMsgType, []byte(serverSendMsg))
		assert.NoError(t, err)

		// must block until connection closes
		bstrm.WaitUntilDisconnect()
		return nil
	}
	lis, err := net.Listen(network, address)
	require.NoError(t, err)

	svc, err = grpcserver.StartGrpcServiceServer(lis, nil, serveStream, nil, time.Minute)
	require.NoError(t, err)
	//defer svc.Stop()

	time.Sleep(time.Millisecond)

	// connect the client and receive the server message
	onClientMessage := func(msgType string, jsonRaw string) {
		msgCount.Add(1)
		assert.Equal(t, serviceCustomMsgType, msgType)
		assert.Equal(t, serverSendMsg, jsonRaw)
	}

	serverURL := fmt.Sprintf("%s://%s", scheme, address)
	cl := grpcclient.NewGrpcServiceClient(clientID, serverURL, nil, time.Minute, onClientMessage)

	err = cl.ConnectWithToken(authToken)
	assert.NoError(t, err) // (dont use require as svc.Stop is not a defer)
	// defer cl.Close()
	// run blocks until the stream is closed
	go func() {
		clientConnectCount.Add(1)
		cl.WaitUntilDisconnect()
		clientConnectCount.Add(1)
	}()

	// some brute force testing on Intel i5-4570S, 2.9GHz:
	// UDS: 1K messages in 5msec; 10K in 44msec; 100K in 310msec; 1M in 3.2 sec
	nrMsg := 10000
	t0 := time.Now()
	for i := 0; i < nrMsg; i++ {
		// slog.Info(fmt.Sprintf("sending %d", i))
		cl.Send("notification", []byte(clientSendMsg))
	}
	dur := time.Since(t0)
	slog.Info(fmt.Sprintf("sent %d messages in %s", nrMsg, dur))
	time.Sleep(time.Millisecond * 10)

	// both client and server side should have received a message
	assert.Equal(t, nrMsg+1, int(msgCount.Load()))

	// graceful shutdown no errors or warndings are expected
	slog.Info("shutting down")
	cl.Close()

	time.Sleep(time.Millisecond * 1)
	svc.Stop()

	// expect a connect and a disconnect
	assert.Equal(t, 2, int(clientConnectCount.Load()))
}

func TestMultipleClients(t *testing.T) {
	// test multiple clients sending messages concurrently
}

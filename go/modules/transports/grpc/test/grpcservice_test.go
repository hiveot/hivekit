// Package grpc_test with test cases to specifically test the gRPC client and server part of the transport
package grpc_test

import (
	"fmt"
	"log/slog"
	"net"
	"os"
	"sync/atomic"
	"testing"
	"time"

	certstest "github.com/hiveot/hivekit/go/modules/certs/test"
	grpclib "github.com/hiveot/hivekit/go/modules/transports/grpc/lib"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thanhpk/randstr"
	"google.golang.org/grpc"
)

// The simplest and fastest is to use UDS sockets
var serverAddress = "/tmp/hivekit/grpc-test.sock"    // host[:port]
var serverNetwork = "unix"                           // unix, tcp, tcp4, tcp6
var clientURL = "unix:///tmp/hivekit/grpc-test.sock" // gRPC doesn't support ipv4, ipv6 schems  (used in client connection URL)

// Using DNS scheme with localhost to avoid gRPC issues with UDS on some platforms (e.g. Windows) and to test the more common TCP use case.
// var serverAddress = "localhost:9988"    // /host[:port]
// var serverNetwork = "tcp"               // unix, tcp, tcp4, tcp6  (used in net.Listen)
// var clientURL = "dns:///localhost:9988" // gRPC doesn't support ipv4, ipv6 schems  (used in client connection URL)

// plain IP address, gRPC doesn't support schemes for ipv4 and ipv6, so simply omit the scheme
// var serverAddress = "127.0.0.1:9988" // /host[:port]
// var serverNetwork = "tcp"            // unix, tcp, tcp4, tcp6  (used in net.Listen)
// var clientURL = serverAddress        // gRPC doesn't support ipv4, ipv6 schems  (used in client connection URL)

var certBundle = certstest.CreateTestCertBundle(utils.KeyTypeED25519)

// TestMain runs a gRPC server
func TestMain(m *testing.M) {
	utils.SetLogging("info", "")
	// slog.Info("------ TestMain of TLSServer_test.go ------")
	// serverAddress = utils.GetOutboundIP("").String()
	// use the localhost interface for testing
	if serverNetwork == "unix" {
		os.MkdirAll("/tmp/hivekit", 0700)
		os.Remove(serverAddress)
	}
	res := m.Run()

	time.Sleep(time.Second)
	os.Exit(res)
}

// TestConnectPing tests creating a UDS or TLS connection with authentication.
func TestConnectPing(t *testing.T) {
	// test connect/disconnect with ping
	t.Logf("---%s---\n", t.Name())
	const clientID = "client1"
	const token = "secret1"
	const serviceName = "service1"

	// setup the server
	lis, err := net.Listen(serverNetwork, serverAddress)
	require.NoError(t, err)

	// TODO test authn
	srv := grpclib.NewGrpcServiceServer(
		lis, certBundle.ServerCert, serviceName, nil, time.Minute)

	err = srv.Start()
	require.NoError(t, err)
	defer srv.Stop()

	// connect with the client
	handleClientMessage := func(raw []byte) {
	}

	cl := grpclib.NewGrpcServiceClient(
		clientURL, certBundle.CaCert, time.Minute, serviceName, handleClientMessage)

	err = cl.ConnectWithToken(clientID, token)
	require.NoError(t, err)

	// test ping
	t0 := time.Now()
	reply, err := cl.Ping("hello world")
	assert.NoError(t, err)
	assert.Equal(t, "hello world", reply)
	d0 := (time.Since(t0) / 10000) * 10000 // rounding to usec
	slog.Info(fmt.Sprintf("Ping performed in %s", d0.String()))
	cl.Close()

	// check closing twice not causing a panic
	assert.NotPanics(t, func() {
		cl.Close()
	})
}

func TestStreamMessages(t *testing.T) {
	// test streaming to server by multiple clients
	t.Logf("---%s---\n", t.Name())

	// bulk message testing
	// some brute force testing on Intel i5-4570S, 2.9GHz:
	// UDS: 100byte->530K msg/sec; 300byte->470K msg/sec; 1K->320K msg/sec; 100K->8K msg/sec
	// TCP: 100byte->480K msg/sec; 300byte->410K msg/sec; 1K->280K msg/sec; 100K->6K msg/sec
	var msgSize = 1000
	// var rxDelay = time.Millisecond * 0
	const serviceName = "service1"
	const streamName = "stream1"

	const clientID = "client1"
	var authToken = "token1"
	var rxCount atomic.Int32
	var clientConnectCount atomic.Int32

	// var serviceCustomMsgType = "serviceMessagetype"
	var clientSendMsg string = string(randstr.Bytes(msgSize))
	var serverSendMsg string = string(randstr.Bytes(msgSize))

	// Handler that receives messages
	handleStream2Message := func(raw []byte) {
		// time.Sleep(rxDelay) // simulate some processing time
		// slog.Info("Receiving msg", "rxCount", rxCount.Load())
		rxCount.Add(1)
	}
	// serveStream := func(clientID, cid string, grpcStream grpcapi.GrpcService_MsgStreamServer) error {
	serveStream2 := func(clientID, cid string, grpcStream grpc.ServerStream) error {
		// todo test authentication?

		// start the send and receive loop
		bstrm := grpclib.NewBufferedStream(
			grpcStream, nil, handleStream2Message, time.Minute)

		// send is dispatched after the stream is
		err := bstrm.Send([]byte(serverSendMsg))
		assert.NoError(t, err)

		// must block until connection closes
		bstrm.WaitUntilDisconnect()
		return nil
	}
	lis, err := net.Listen(serverNetwork, serverAddress)
	require.NoError(t, err)

	// certBundle.ServerCert = nil
	// certBundle.CaCert = nil

	srv := grpclib.NewGrpcServiceServer(lis, certBundle.ServerCert, serviceName, nil, time.Minute)
	srv.CreateStream(streamName, serveStream2)

	err = srv.Start()
	require.NoError(t, err)

	// connect the client and receive the server message
	onClientMessage := func(raw []byte) {
		// onClientMessage := func(msgType string, raw []byte) {
		rxCount.Add(1)
		// assert.Equal(t, serviceCustomMsgType, msgType)
		rxMsg := utils.DecodeAsString(raw, 0)
		// rxMsg := string(raw)
		assert.Equal(t, serverSendMsg, rxMsg)
	}
	cl := grpclib.NewGrpcServiceClient(
		clientURL, certBundle.CaCert, time.Minute, serviceName, onClientMessage)

	err = cl.ConnectWithToken(clientID, authToken)
	assert.NoError(t, err) // (dont use require as svc.Stop is not a defer)

	_, err = cl.ConnectStream(streamName)
	require.NoError(t, err) // (dont use require as svc.Stop is not a defer)

	// defer cl.Close()
	// run blocks until the stream is closed
	go func() {
		clientConnectCount.Add(1)
		cl.WaitUntilDisconnect(streamName)
		clientConnectCount.Add(1)
	}()

	// send messages until a second has passed
	t0 := time.Now()
	t1 := t0.Add(time.Millisecond * 1000)
	var nrMsg int
	for nrMsg = 1; ; nrMsg++ {
		// slog.Info(fmt.Sprintf("sending %d", i))
		err = cl.Send(streamName, []byte(clientSendMsg))
		if time.Now().After(t1) {
			break
		}
		assert.NoError(t, err)
	}
	// give the receiver time to catch up
	time.Sleep(time.Millisecond * 1000)

	slog.Info(fmt.Sprintf("sent %d messages/sec;", nrMsg))

	// both client and server side should have received a message
	assert.Equal(t, nrMsg+1, int(rxCount.Load()), "Client did not receive all messages. Some got lost!?")

	// graceful shutdown no errors or warndings are expected
	slog.Info("shutting down")
	cl.Close()

	time.Sleep(time.Millisecond * 1)
	srv.Stop()

	// expect a connect and a disconnect
	assert.Equal(t, 2, int(clientConnectCount.Load()))
}

func TestMultipleClients(t *testing.T) {
	// test multiple clients sending messages concurrently
}

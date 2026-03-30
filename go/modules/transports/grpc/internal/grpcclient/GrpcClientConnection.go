package grpcclient

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	grpcapi "github.com/hiveot/hivekit/go/modules/transports/grpc/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const ClientMsgChanSize = 30

// Grpc messaging client
type GrpcServiceClient struct {
	// this channel is used to signal the read/write loops to exit
	cancelChan chan struct{}

	connectionID string

	// callback for connection status changes
	connHandler func(connected bool, err error)

	// URL to connect to. see also https://github.com/grpc/grpc/blob/master/doc/naming.md
	// unix:///path/to/socket
	// ipv4://address:[port][,address[:port]]
	// dns://address:[port]
	connectURI string

	// conn         net.Conn
	grpcConn          *grpc.ClientConn
	grpcServiceClient grpcapi.GrpcServiceClient // interface
	//
	msgStream       grpcapi.GrpcService_MsgStreamClient // interface
	msgStreamCancel func()

	// callback for incoming messages
	msgHandler func(msgType string, rawJson string)

	// mutex for controlling writing and closing
	mux sync.RWMutex

	// retryOnDisconnect atomic.Bool
	isConnected atomic.Bool
	// read stream context cancellation
	rxCtxCancelFn func()

	// the send channel with buffer to force sequential sending
	sendChan chan *grpcapi.GrpcMsg
	// write respTimeout
	respTimeout time.Duration
}

// when the client disconnects, we want to make sure that the read loop exits gracefully and
// that all pending response handlers are notified of the disconnection.
func (sc *GrpcServiceClient) cancelSafely() {
	if !sc.isConnected.Load() {
		return
	}
	sc.isConnected.Store(false)
	close(sc.cancelChan)
	close(sc.sendChan)
}

func (cl *GrpcServiceClient) readLoop() (err error) {

	// run the stream read loop in the background until it is cancelled
	go func() {
		for {
			result, err := cl.msgStream.Recv()
			if err == io.EOF {
				slog.Warn("Client readLoop: grpc stream read loop closed due to EOF")
				break
			} else if err != nil {
				slog.Warn("Client readLoop: Recv error", "err", err.Error())
				break
			}

			slog.Info("client readLoop:received message:" + result.MsgType)
			// this should prolly run in the background
			cl.msgHandler(result.MsgType, result.JsonPayload)
		}
		slog.Info("client readLoop: readLoop ended")
		cl.cancelSafely()
	}()
	return err
}

// Send loop using the send channel to ensure sequential delivery of messages.
// grpc streams do not support concurrent sending.
func (sc *GrpcServiceClient) sendLoop() {
	for msg := range sc.sendChan {
		if err := sc.msgStream.Send(msg); err != nil {
			break
		}
	}
	slog.Info("client sendLoop: sendLoop ended")
	sc.cancelSafely() // in	case client disconnected
}

// Close disconnects
func (cl *GrpcServiceClient) Close() {
	slog.Info("client Close: ending read/write loops")
	if cl.grpcConn != nil {
		cl.grpcConn.Close()
	}
	if cl.msgStreamCancel != nil {
		cl.msgStreamCancel()
		cl.msgStreamCancel = nil
	}
}

// Initiate a connection to the grpc server
// The actually connection is established on first request.
func (cl *GrpcServiceClient) Connect() (err error) {
	if cl.grpcConn != nil {
		cl.grpcConn.Close()
	}
	// cl.conn, err = net.Dial("unix", cl.socketPath)
	var opts []grpc.DialOption

	// creds, err := credentials.NewClientTLSFromFile("cert/server.crt", "")
	dialOpt := grpc.WithTransportCredentials(insecure.NewCredentials())
	opts = append(opts, dialOpt)

	cl.grpcConn, err = grpc.NewClient(cl.connectURI, opts...)
	if err != nil {
		return err
	}
	cl.grpcServiceClient = grpcapi.NewGrpcServiceClient(cl.grpcConn)

	ctx, cancelFn := context.WithCancel(context.Background())
	cl.msgStream, err = cl.grpcServiceClient.MsgStream(ctx)
	cl.msgStreamCancel = cancelFn

	return err
}

// // GetConnectionID returns the client's connection details
func (cl *GrpcServiceClient) GetConnectionID() string {

	// udc, found := cl.conn.(*net.UnixConn)
	// if found {
	// 	fd, _ := udc.File()
	// 	fdName := fd.Name()
	// 	fdfd := fd.Fd()
	// 	idText := fmt.Sprintf("%s [%d]", fdName, fdfd)
	// 	return idText
	// }
	// tcp, found := cl.conn.(*net.TCPConn)
	// if found {
	// 	ra := tcp.RemoteAddr()
	// 	fd, _ := tcp.File()
	// 	fdfd := fd.Fd()
	// 	idText := fmt.Sprintf("%s [%d]", ra, fdfd)
	// 	return idText
	// }
	return "closed"
	// return cl.connectionID
}

func (cl *GrpcServiceClient) Ping(pingText string) (reply string, err error) {
	pingMsg := &grpcapi.PingMsg{
		Text: pingText,
	}
	ctx, cancelFn := context.WithTimeout(context.Background(), time.Minute)
	defer cancelFn()
	replyMsg, err := cl.grpcServiceClient.Ping(ctx, pingMsg)
	if err != nil {
		return "", err
	}
	return replyMsg.Text, nil
}

// Send a message to the server
func (cl *GrpcServiceClient) Send(msgType string, jsonPayload string) {
	grpcMsg := &grpcapi.GrpcMsg{
		MsgType:     msgType,
		JsonPayload: jsonPayload,
	}

	// streamClient, err := cl.grpcServiceClient.StreamGrpcMsg( context.Background(), nil)
	cl.msgStream.Send(grpcMsg)
}

// Run starts processing a message stream from the client.
// This returns when the stream is closed.
func (sc *GrpcServiceClient) Run() {
	// default buffer size. TODO:flow control
	sc.sendChan = make(chan *grpcapi.GrpcMsg, ClientMsgChanSize)
	sc.cancelChan = make(chan struct{})
	sc.isConnected.Store(true)
	go sc.readLoop()
	go sc.sendLoop()
	<-sc.cancelChan
}

// Write a message over the gUDS socket connection
// func (cl *GrpcClient) WriteMessage(data any) error {

// 	if cl.conn == nil {
// 		err := fmt.Errorf("WriteMessage: Can't send. Not connected")
// 		return err
// 	}
// 	// socket does not allow concurrent writes
// 	cl.mux.Lock()
// 	defer cl.mux.Unlock()

// 	// TODO: what is a common serialization format for UDS?
// 	// how do UDS messages use message boundaries
// 	raw, err := jsoniter.Marshal(data)
// 	if err != nil {
// 		return err
// 	}
// 	deadline := time.Now().Add(cl.timeout)
// 	cl.conn.SetWriteDeadline(deadline)

// 	n, err := cl.conn.Write(raw)
// 	if n != len(raw) {
// 		err = fmt.Errorf("Write: written only '%d' of '%d' bytes", n, len(raw))
// 		slog.Error(err.Error())
// 	}
// 	return err
// }

func NewGrpcServiceClient(connectURI string,
	respTimeout time.Duration,
	msgHandler func(msgType string, jsonRaw string),
	ch func(connected bool, err error),
) *GrpcServiceClient {

	cl := &GrpcServiceClient{
		connHandler: ch,
		connectURI:  connectURI,
		msgHandler:  msgHandler,
		respTimeout: respTimeout,
	}
	return cl
}

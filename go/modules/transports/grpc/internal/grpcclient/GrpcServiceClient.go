package grpcclient

import (
	"context"
	"log/slog"
	"sync"
	"time"

	grpcapi "github.com/hiveot/hivekit/go/modules/transports/grpc/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
)

const ClientMsgChanSize = 30

// Grpc messaging client
type GrpcServiceClient struct {

	// buffered stream wrapper around the protobuf stream.
	bufStream *GrpcBufferedStream

	// this channel is used to signal the read/write loops to exit
	// cancelChan chan struct{}

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
	// msgStream       grpcapi.GrpcService_MsgStreamClient // interface
	msgStreamCancel func()

	// callback for incoming messages
	recvHandler func(msgType string, rawJson string)

	// mutex for controlling writing and closing
	mux sync.RWMutex

	// retryOnDisconnect atomic.Bool
	// isConnected atomic.Bool
	// read stream context cancellation
	// rxCtxCancelFn func()

	// the send channel with buffer to force sequential sending
	// sendChan chan *grpcapi.GrpcMsg
	// write respTimeout
	respTimeout time.Duration
}

// when the client disconnects, we want to make sure that the read loop exits gracefully and
// that all pending response handlers are notified of the disconnection.
// func (cl *GrpcServiceClient) cancelSafely() {
// 	if !cl.isConnected.Load() {
// 		return
// 	}
// 	cl.isConnected.Store(false)
// 	close(cl.cancelChan)

// 	if cl.grpcConn != nil {
// 		cl.grpcConn.Close()
// 	}
// 	if cl.msgStreamCancel != nil {
// 		cl.msgStreamCancel()
// 		cl.msgStreamCancel = nil
// 	}
// }

// func (cl *GrpcServiceClient) readLoop() (err error) {

// 	// run the stream read loop in the background until it is cancelled
// 	go func() {
// 		for {
// 			slog.Info("readLoop: Recv enter")
// 			result, err := cl.msgStream.Recv()
// 			slog.Info("readLoop: Recv exit", "err", err)
// 			if err == io.EOF {
// 				// server closed the connection?
// 				slog.Warn("Client readLoop: grpc stream read loop closed due to EOF")
// 				break
// 			} else if err != nil {
// 				slog.Warn("Client readLoop: Recv error", "err", err.Error())
// 				break
// 			}

// 			slog.Info("client readLoop:received message:" + result.MsgType)
// 			// this should prolly run in the background
// 			cl.recvHandler(result.MsgType, result.JsonPayload)
// 		}
// 		slog.Info("client readLoop: readLoop ended")
// 		cl.cancelSafely()
// 	}()
// 	return err
// }

// Send loop using the send channel to ensure sequential delivery of messages.
// grpc streams do not support concurrent sending.
// func (cl *GrpcServiceClient) sendLoop() {
// 	for msg := range cl.sendChan {
// 		if err := cl.msgStream.Send(msg); err != nil {
// 			slog.Info("client sendLoop: error", "err", err.Error())
// 			break
// 		}
// 	}
// 	slog.Info("client sendLoop: sendLoop ended")
// 	// sc.cancelSafely() // in	case client disconnected
// }

// Close disconnects
func (cl *GrpcServiceClient) Close() {
	cl.bufStream.Close()
	cl.grpcConn.Close()
	// cl.cancelSafely()
	slog.Info("client Close: ending read/write loops")
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

	// ctx, cancelFn := context.WithCancel(context.Background())
	// cl.msgStream, err = cl.grpcServiceClient.MsgStream(ctx)
	// cl.msgStreamCancel = cancelFn

	// TODO: use buffered stream for sending
	ctx, cancelFn := context.WithCancel(context.Background())
	cl.msgStreamCancel = cancelFn
	msgStream, err := cl.grpcServiceClient.MsgStream(ctx)
	cl.bufStream = NewGrpcBufferedStream(msgStream, cl.recvHandler, cl.respTimeout)

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
	return "todo"
	// return cl.connectionID
}
func (cl *GrpcServiceClient) IsConnected() bool {
	return cl.bufStream.IsConnected()
}

func (cl *GrpcServiceClient) Ping(pingText string) (reply string, err error) {
	ctx, cancelFn := context.WithTimeout(context.Background(), cl.respTimeout)
	defer cancelFn()
	replyMsg, err := cl.grpcServiceClient.Ping(ctx, &emptypb.Empty{})
	if err != nil {
		return "", err
	}
	return replyMsg.Text, nil
}

// Send a message to the server
func (cl *GrpcServiceClient) Send(msgType string, jsonPayload string) {
	cl.bufStream.Send(msgType, jsonPayload)
	// grpcMsg := &grpcapi.GrpcMsg{
	// 	MsgType:     msgType,
	// 	JsonPayload: jsonPayload,
	// }
	// cl.msgStream.Send(grpcMsg)
}

// Run starts processing a message stream from the client.
// This returns when the stream is closed.
func (cl *GrpcServiceClient) Run() {

	cl.bufStream.WaitUntilDisconnect()

	// default buffer size. TODO:flow control
	// cl.sendChan = make(chan *grpcapi.GrpcMsg, ClientMsgChanSize)
	// cl.cancelChan = make(chan struct{})
	// cl.isConnected.Store(true)
	// go cl.readLoop()
	// go cl.sendLoop()
	// <-cl.cancelChan
}

func NewGrpcServiceClient(connectURI string,
	respTimeout time.Duration,
	msgHandler func(msgType string, jsonRaw string),
	ch func(connected bool, err error),
) *GrpcServiceClient {

	cl := &GrpcServiceClient{
		connHandler: ch,
		connectURI:  connectURI,
		recvHandler: msgHandler,
		respTimeout: respTimeout,
	}
	return cl
}

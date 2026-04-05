package grpcclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/modules/transports/grpc/internal"
	"github.com/hiveot/hivekit/go/modules/transports/grpc/internal/grpcserver"
	"github.com/teris-io/shortid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/encoding"
)

const ClientMsgChanSize2 = 30

// Stream info that matches the server
// var streamInfo = grpc.StreamDesc{

// 	// a not very original name for this stream
// 	StreamName: "MsgStream",
// 	// this handler serves the stream with StreamName name.
// 	// Handler: func(srv interface{}, stream grpc.ServerStream) error {
// 	// 	return srv.(*GrpcServiceServer2).MsgStream(stream)
// 	// },
// 	ServerStreams: true,
// 	ClientStreams: true,
// }

// Grpc messaging client.
// This uses the BufferedStream for sending and receiving messages on the GRPC stream
// connection. The buffer size is currently fixed to 30 messages, which should be
// sufficient for most use cases.
//
// This also implements the PerTransportBundle PerRPCCredentials interface
type GrpcServiceClient struct {
	// pass auth token to GetRequestMetadata
	authToken string

	// buffered stream wrapper around the protobuf stream.
	bufStream *internal.BufferedStream

	// caCert in case the connectURL is an ip connection address
	caCert *x509.Certificate

	// clientID and connectionID to include in the grpc metadata.
	clientID     string
	connectionID string

	// URL to connect to. see also https://github.com/grpc/grpc/blob/master/doc/naming.md
	// unix:///path/to/socket
	// ipv4://address:[port][,address[:port]]
	// dns://address:[port]
	connectURL string

	// conn         net.Conn
	grpcConn *grpc.ClientConn
	// grpcServiceClient grpcapi.GrpcServiceClient // interface
	//

	// message stream context cancellation
	msgStreamCancel func()

	// mutex for controlling writing and closing
	mux sync.RWMutex

	// ping from the gRPC protobuf definition
	pingHandler func(context.Context, any) (reply string, err error)

	// callback for incoming messages
	// the grpc codec determines the type of any
	recvHandler func(rawMsg []byte)

	respTimeout time.Duration

	retryOnDisconnect atomic.Bool
}

// Close disconnects
func (cl *GrpcServiceClient) Close() {
	cl.mux.Lock()
	defer cl.mux.Unlock()
	if cl.bufStream.IsConnected() {
		cl.bufStream.Close()
	}
	if cl.msgStreamCancel != nil {
		cl.msgStreamCancel()
		cl.msgStreamCancel = nil
	}
	if cl.grpcConn != nil {
		cl.grpcConn.Close()
		cl.grpcConn = nil
	}
	slog.Info("client Close: ending read/write loops")
}

// Initiate a connection to the grpc server
func (cl *GrpcServiceClient) ConnectWithToken(clientID string, authToken string) (err error) {
	cl.mux.Lock()
	defer cl.mux.Unlock()
	cl.clientID = clientID

	// fail if a connection already exists
	if cl.grpcConn != nil {
		return fmt.Errorf("Connect: A connection already exists. Close first.")
	}

	dialOpts := []grpc.DialOption{}

	// If a CA certificate is set the use Transport credentials
	if cl.caCert != nil {
		// configure the TransportCredentials dial option
		var clientCertList []tls.Certificate
		// TODO: support client cert auth

		caCertPool := x509.NewCertPool()
		caCertPool.AddCert(cl.caCert)

		tlsConfig := &tls.Config{
			RootCAs:            caCertPool,
			InsecureSkipVerify: false,
			Certificates:       clientCertList,
		}
		tlsCreds := credentials.NewTLS(tlsConfig)
		tlsCredOpt := grpc.WithTransportCredentials(tlsCreds)
		dialOpts = append(dialOpts, tlsCredOpt)
	} else {
		nocreds := grpc.WithTransportCredentials(insecure.NewCredentials())
		dialOpts = append(dialOpts, nocreds)
	}

	// this client implements the PerRPCCredentials interface
	// see: GetRequestMetadata and RequireTransportSecurity
	cl.authToken = authToken
	rpcCredOpt := grpc.WithPerRPCCredentials(cl)
	dialOpts = append(dialOpts, rpcCredOpt)

	// use the custom codec instead of protobuf
	// this is a codec per-call, hence use WithDefaultCallOptions
	encoding.RegisterCodec(internal.RawCodec{})
	codecOption := grpc.WithDefaultCallOptions(grpc.CallContentSubtype("rawcodec"))
	dialOpts = append(dialOpts, codecOption)

	cl.grpcConn, err = grpc.NewClient(cl.connectURL, dialOpts...)
	if err != nil {
		slog.Error("Connect: NewClient failed", "err", err.Error())
		return err
	}
	// grpcServiceClient := grpcapi.NewGrpcServiceClient(cl.grpcConn)
	// ctx, cancelFn := context.WithCancel(context.Background())
	// cl.msgStreamCancel = cancelFn
	// msgStream, err := grpcServiceClient.MsgStream(ctx)

	// Create the messaging stream
	ctx, cancelFn := context.WithCancel(context.Background())
	cl.msgStreamCancel = cancelFn
	opts := []grpc.CallOption{}

	// this returns a grpc.ClientStream
	// NOTE: streamInfo must match the server
	// var streamInfo = grpc.StreamDesc{
	// 	StreamName:    grpcapi.GrpcTransportStreamName, // stream name must match the server
	// 	ServerStreams: true,
	// 	ClientStreams: true,
	// }
	// FIXME: the client requires stream name below
	streamdesc := &grpcserver.ServiceDesc.Streams[0]
	streamName := "/grpcapi.GrpcService/MsgStream"
	grpcStream, err := cl.grpcConn.NewStream(ctx,
		streamdesc, streamName, opts...)

	if err != nil {
		slog.Error("Connect: MsgStream failed", "err", err.Error())
		return err
	}

	// This adds Send and Receive methods to the stream
	// msgStream := NewGrpcServiceStreamClient2(grpcStream)
	// var blob []byte
	// err = grpcStream.RecvMsg(&blob)
	// if err != nil {
	// 	slog.Error("read stream failed ", err, err.Error())
	// 	return err
	// }

	// use buffered stream for sending and receiving
	cl.bufStream = internal.NewBufferedStream(grpcStream, cl.recvHandler, cl.respTimeout)

	return err
}

// // GetConnectionID returns the client's connection details
func (cl *GrpcServiceClient) GetConnectionID() string {
	return cl.connectionID
}

func (cl *GrpcServiceClient) IsConnected() bool {
	return cl.bufStream != nil && cl.bufStream.IsConnected()
}

func (cl *GrpcServiceClient) Ping() (reply string, err error) {
	ctx, cancelFn := context.WithTimeout(context.Background(), cl.respTimeout)
	defer cancelFn()
	opts := []grpc.CallOption{}
	in := []byte("ping")
	var out []byte
	err = cl.grpcConn.Invoke(ctx, "/grpcapi.GrpcService/ping", in, &out, opts...)
	// replyMsg, err := cl.grpcServiceClient.Ping(ctx, text)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// PerRPCCredentials:GetRequestMetadata
func (cl *GrpcServiceClient) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	// bearer authentication
	return map[string]string{
		"authorization":               "bearer " + cl.authToken,
		transports.ClientIDContextID:  cl.clientID,
		transports.ClientCIDContextID: cl.connectionID,
	}, nil
}

// PerRPCCredentials:RequireTransportSecurity
// FIXME: support for TLS certificate only when using tcp connections, not when using UDS
func (cl *GrpcServiceClient) RequireTransportSecurity() bool { return false }

// Send a message to the server
func (cl *GrpcServiceClient) Send(rawMsg []byte) (err error) {
	err = cl.bufStream.Send(rawMsg)
	return err
}

// WaitUntilDisconnect waits until the client connection is closed.
func (cl *GrpcServiceClient) WaitUntilDisconnect() {
	cl.bufStream.WaitUntilDisconnect()
}

// Create a client for the GRPC transport
// caCert is optional for use with tcp sockets
func NewGrpcServiceClient(
	connectURI string, caCert *x509.Certificate,
	respTimeout time.Duration,
	msgHandler func(rawMsg []byte),
) *GrpcServiceClient {

	cl := &GrpcServiceClient{
		caCert:       caCert,
		connectionID: shortid.MustGenerate(),
		connectURL:   connectURI,
		recvHandler:  msgHandler,
		respTimeout:  respTimeout,
	}
	return cl
}

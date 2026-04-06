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
	"github.com/teris-io/shortid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/encoding"
)

const ClientMsgChanSize2 = 30

// Grpc messaging client.
// This uses the BufferedStream for sending and receiving messages on the GRPC stream
// connection. The buffer size is currently fixed to 30 messages, which should be
// sufficient for most use cases.
//
// Intended to implement the boilerplate of setting up a client side stream without
// the use of protobuf. The only thing needed is service name, stream name and the
// proper codec initialized by the application.
// This uses the json codec. Might consider sticking to the default protobuf encoder?
//
// # This also implements the PerTransportBundle PerRPCCredentials interface
//
// This client is initially made for the transport module but could be generalized for
// other purposes such as media streaming.
type GrpcServiceClient struct {
	// pass auth token to GetRequestMetadata
	authToken string

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

	// message stream context cancellation
	// msgStreamCancel func()

	// mutex for controlling writing and closing
	mux sync.RWMutex

	// ping from the gRPC protobuf definition
	pingHandler func(context.Context, any) (reply string, err error)

	// callback for incoming messages
	// the grpc codec determines the type of any
	recvHandler func(rawMsg []byte)

	respTimeout time.Duration

	retryOnDisconnect atomic.Bool

	// The GRPC server service description
	serviceDesc grpc.ServiceDesc

	// buffered stream wrapper around the gRPC streams by stream name (from serviceDesc)
	streams map[string]*internal.BufferedStream
}

// Close disconnects
func (cl *GrpcServiceClient) Close() {
	cl.mux.Lock()
	defer cl.mux.Unlock()
	for _, s := range cl.streams {
		if s.IsConnected() {
			s.Close()
		}
	}
	if cl.grpcConn != nil {
		cl.grpcConn.Close()
		cl.grpcConn = nil
	}
	slog.Info("client Close: ending read/write loops")
}

// Initiate a connection to the grpc server.
// The owner must open the streams it wasnt to use using 'ConnectStream'.
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

	// use the custom JSON codec instead of the default protobuf.
	// this is a codec per-call, hence use WithDefaultCallOptions
	// TODO: for use with http2 see also https://github.com/grpc/grpc/blob/master/doc/PROTOCOL-HTTP2.md#requests
	// which seems to want base64 encoding. Not a concern right now.
	codec := internal.JsonCodec{}
	encoding.RegisterCodec(codec)

	codecOption := grpc.WithDefaultCallOptions(grpc.CallContentSubtype(codec.Name()))
	dialOpts = append(dialOpts, codecOption)

	cl.grpcConn, err = grpc.NewClient(cl.connectURL, dialOpts...)
	if err != nil {
		slog.Error("Connect: NewClient failed", "err", err.Error())
		return err
	}

	return err
}

// ConnectStream connects to a server stream
// This returns a buffered stream.
// the stream is added to the 'streams' map.
// 'name' is the registered server stream name (eg, 'notification', or 'request/response')
//
// This returns the buffered stream or an error if failed
func (cl *GrpcServiceClient) ConnectStream(name string) (*internal.BufferedStream, error) {

	// Create the messaging stream
	ctx, cancelFn := context.WithCancel(context.Background())
	opts := []grpc.CallOption{}

	// Open the stream
	// the stream name is the service name / stream name
	// client and server must use the same service name and stream name
	streamDesc := &grpc.StreamDesc{
		ServerStreams: true,
		ClientStreams: true,
	}
	// Note, two issues here:
	// 1. the stream name in streaemDesc isn't used by grpc
	// 2. the stream name must be prefixed with the service name: full name=service/stream
	serviceStreamName := cl.serviceDesc.ServiceName + "/" + name
	stream, err := cl.grpcConn.NewStream(ctx,
		streamDesc, serviceStreamName, opts...)

	if err != nil {
		slog.Error("Connect: MsgStream failed", "err", err.Error())
		cancelFn()
		return nil, err
	}

	// use buffered stream for sending and receiving
	bufferedStream := internal.NewBufferedStream(stream, cancelFn, cl.recvHandler, cl.respTimeout)

	cl.mux.Lock()
	defer cl.mux.Unlock()
	cl.streams[name] = bufferedStream
	return bufferedStream, nil
}

// // GetConnectionID returns the client's connection details
func (cl *GrpcServiceClient) GetConnectionID() string {
	return cl.connectionID
}

// GetStream returns the stream with the given name or an error if not found
func (cl *GrpcServiceClient) GetStream(name string) (*internal.BufferedStream, error) {
	cl.mux.RLock()
	defer cl.mux.RUnlock()
	strm, _ := cl.streams[name]
	if strm == nil {
		return nil, fmt.Errorf("Stream '%s' not found", name)
	}
	return strm, nil
}

// Test if the stream with the given name is connected
func (cl *GrpcServiceClient) IsConnected(name string) bool {
	strm, err := cl.GetStream(name)
	if err != nil {
		return false
	}
	return strm.IsConnected()
}

// a simple ping test
// this echos the input or returns 'pong' if none is provided
func (cl *GrpcServiceClient) Ping(input string) (reply string, err error) {
	ctx, cancelFn := context.WithTimeout(context.Background(), cl.respTimeout)
	defer cancelFn()
	opts := []grpc.CallOption{}
	in := input
	// TODO: the service name
	serviceMethodName := cl.serviceDesc.ServiceName + "/ping"
	err = cl.grpcConn.Invoke(ctx, serviceMethodName, in, &reply, opts...)
	// replyMsg, err := cl.grpcServiceClient.Ping(ctx, text)
	if err != nil {
		return "", err
	}
	return reply, nil
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
// name selects the stream name
func (cl *GrpcServiceClient) Send(name string, rawMsg []byte) (err error) {
	strm, err := cl.GetStream(name)
	if err == nil {
		err = strm.Send(rawMsg)
	}
	return err
}

// WaitUntilDisconnect waits until the client streams are is closed.
// This returns an error if the stream name isnt found
func (cl *GrpcServiceClient) WaitUntilDisconnect(name string) error {
	strm, err := cl.GetStream(name)
	if err == nil {
		strm.WaitUntilDisconnect()
	}
	return err
}

// Create a client for the GRPC transport
//
// The serviceName is provided by the application and must match the server.
// Use ConnectStream(name) to connect to individual server streams.
// caCert is optional for use with tcp sockets
func NewGrpcServiceClient(
	connectURI string, caCert *x509.Certificate,
	respTimeout time.Duration,
	serviceName string,
	msgHandler func(rawMsg []byte),
) *GrpcServiceClient {

	// generate the client side service description
	// the service name should be provided by the user
	serviceDesc := grpc.ServiceDesc{
		ServiceName: serviceName, // grpcapi.GrpcTransportServiceName,
		// HandlerType: not needed client side
		Methods: []grpc.MethodDesc{},
		Streams: []grpc.StreamDesc{},
	}

	cl := &GrpcServiceClient{
		caCert:       caCert,
		connectionID: shortid.MustGenerate(),
		connectURL:   connectURI,
		recvHandler:  msgHandler,
		respTimeout:  respTimeout,
		serviceDesc:  serviceDesc,
		streams:      make(map[string]*internal.BufferedStream),
	}
	return cl
}

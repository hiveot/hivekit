package grpclib

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"log/slog"
	"net"
	"strings"
	"time"

	"github.com/hiveot/hivekit/go/modules/transports"

	"github.com/hiveot/hivekit/go/msg"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/encoding"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const PingMethodName = "ping"

// GRPC server handler of protobuf defined methods.
// This currently only implements the Ping and MsgStream methods.
type GrpcServiceServer struct {
	grpcAuthn *GrpcAuthenticator

	// the underlying GRPC server
	grpcServer *grpc.Server

	// The network listener, unix or tcp socket
	lis net.Listener

	// callback for serving a new notification stream
	// serveNotifStream func(clientID string, cid string, grpcStream grpc.ServerStream) error

	// callback for serving a new request/response stream
	serveReqRespStream func(clientID string, cid string, grpcStream grpc.ServerStream) error

	// how long to wait for a response after sending a request
	respTimeout time.Duration

	// todo
	rnrChan *msg.RnRChan

	// service description for methods and streams
	serviceDesc grpc.ServiceDesc

	// optional TLS certificate when using TCP instead of UDS
	tlsCert *tls.Certificate
}

// authenticate a new stream connection
func (srv *GrpcServiceServer) streamInterceptor(
	srv2 interface{},
	ss grpc.ServerStream,
	info *grpc.StreamServerInfo,
	handler grpc.StreamHandler) (err error) {

	if srv.grpcAuthn != nil {
		_, _, err := srv.grpcAuthn.Authenticate(ss.Context())
		if err != nil {
			slog.Error("streamInterceptor: Unauthenticated")

			return status.Errorf(codes.Unauthenticated, "Unauthenticated: %s", err.Error())
		}
	}
	return handler(srv2, ss)
}

// unaryInterceptor calls authenticateClient with current context
func (srv *GrpcServiceServer) unaryInterceptor(
	ctx context.Context, req interface{}, info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler) (interface{}, error) {

	if srv.grpcAuthn != nil {
		_, _, err := srv.grpcAuthn.Authenticate(ctx)
		if err != nil {
			return nil, status.Errorf(codes.Unauthenticated, "Unauthenticated: %s", err.Error())
		}
	}
	return handler(ctx, req)
}

// Create a new stream to listen on
// Note that the handler is called with the raw grpc stream, which is not concurrent safe.
//
// Use NewBufferedStream(grpcStream) to create a buffered concurrently safe
// connection for this stream. This buffered stream has send and receive methods.
func (srv *GrpcServiceServer) CreateStream(
	name string, handler func(clientID, cid string, grpcStream grpc.ServerStream) error) error {

	srv.serviceDesc.Streams = append(srv.serviceDesc.Streams, grpc.StreamDesc{
		StreamName: name,
		// this handler serves the stream with StreamName name.
		Handler: func(_ interface{}, stream grpc.ServerStream) error {
			clientID, cid, err := srv.GetRequestParams(stream.Context())
			if err != nil {
				return err
			}
			err = handler(clientID, cid, stream)
			return err
			// return srv.(*GrpcServiceServer).NewMsgStream(name, stream)
		},
		ServerStreams: true,
		ClientStreams: true,
	})
	return nil
}

// Return the request parameters from the grpc context
func (srv *GrpcServiceServer) GetRequestParams(ctx context.Context) (
	clientID string, cid string, err error) {

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		slog.Error("GetRequestParams: missing auth metadata context")
		return "", "", fmt.Errorf("missing metadata")
	}
	clientID = strings.Join(md[transports.ClientIDContextID], "")
	cid = strings.Join(md[transports.ClientCIDContextID], "")
	return clientID, cid, err
}

// Handler of ping message returns pong
func (srv *GrpcServiceServer) Ping(ctx context.Context, input string) (result string, err error) {

	clientID, cid, err := srv.GetRequestParams(ctx)
	_ = err
	log.Printf("GrpcServiceServer.Ping: ping received from clientID '%s', cid='%s'", clientID, cid)
	result = input
	if input == "" {
		result = "pong"
	}
	return result, nil
}

// register the stream server and start listening
func (srv *GrpcServiceServer) Start() error {

	var grpcServer *grpc.Server
	var opts = make([]grpc.ServerOption, 0)

	// Create the TLS credentials
	// creds, err := credentials.NewServerTLSFromFile(certFile, keyFile)
	if srv.tlsCert != nil {
		creds := credentials.NewServerTLSFromCert(srv.tlsCert)
		// Create an array of gRPC options with the credentials
		opts = append(opts, grpc.Creds(creds))
	}

	// auth and stuff
	opts = append(opts, grpc.UnaryInterceptor(srv.unaryInterceptor))
	opts = append(opts, grpc.StreamInterceptor(srv.streamInterceptor))

	grpcServer = grpc.NewServer(opts...)

	// ServiceDesc.ServiceName = serviceName
	grpcServer.RegisterService(&srv.serviceDesc, nil) // srv)

	// grpcapi.RegisterGrpcServiceServer(grpcServer, srv)
	srv.grpcServer = grpcServer

	// start the server
	slog.Info("StartGrpcServer: starting  gRPC server", slog.String("Address", srv.lis.Addr().String()))
	go func() {
		if err := grpcServer.Serve(srv.lis); err != nil {
			slog.Error("StartGrpcServer: failed to serve", "err", err.Error())
		}
	}()
	return nil
}

// graceful stop of the server
func (srv *GrpcServiceServer) Stop() {
	if srv.grpcServer != nil {
		srv.grpcServer.GracefulStop()
		srv.grpcServer = nil
	}
	srv.lis.Close()
}

// Create the GRPC server, register a ping handler and listen for incoming connections.
//
// Example usage:
//
// > lis, err := net.Listen("unix", "/var/app.sock")
// > srv := NewGrpcServiceServer(lis, nil, "service1", nil, time.Minute)
// or:
// > lis, err := net.Listen("tcp", ":8899")
// > srv := NewGrpcServiceServer(lis, tlsCert, "service1", authn, time.Minute)
// then:
// > srv.Start()
// > srv.CreateStream("stream1", onStream1)
//
//	lis is the network to listen on. This will be closed when the server is stopped.
//	tlsCert is the TLS certificate to use for secure connections, or nil for insecure
//	serviceName is the service name the streams are reachable under
//	grpcAuthn is the grpc connection authenticator
//	respTimeout is the messaging timeout
func NewGrpcServiceServer(
	lis net.Listener,
	tlsCert *tls.Certificate,
	serviceName string,
	grpcAuthn *GrpcAuthenticator,
	respTimeout time.Duration,
) *GrpcServiceServer {

	srv := &GrpcServiceServer{
		grpcAuthn:   grpcAuthn,
		lis:         lis,
		respTimeout: respTimeout,
		// serveNotifStream: serveHandler,
		tlsCert: tlsCert,
	}

	// This codec handles byte array and strings without conversion and uses JSON for anything else.
	// TODO: when used with http2 this might need base64 encoding instead
	// See also https://github.com/grpc/grpc/blob/master/doc/PROTOCOL-HTTP2.md#requests
	// !The incoming request content-type header must match the codec name.
	// or force it using grpc.ForceServerCodec()
	// note: registration applies to all client and servers
	encoding.RegisterCodec(JsonCodec{})

	// var _ IGrpcServiceServer = srv // interface check

	// create a service description with a default ping method
	// streams can be added with 'CreateStream' before Start is called.
	srv.serviceDesc = grpc.ServiceDesc{
		ServiceName: serviceName,
		// HandlerType: (*IGrpcServiceServer)(nil),
		Methods: []grpc.MethodDesc{
			{
				MethodName: PingMethodName,
				Handler: func(_ interface{}, ctx context.Context,
					dec func(any) error,
					interceptor grpc.UnaryServerInterceptor) (any, error) {

					var input string
					err := dec(&input)
					if err == nil {
						return srv.Ping(ctx, input)
					}
					return input, err
				},
			},
		},
	}
	return srv
}

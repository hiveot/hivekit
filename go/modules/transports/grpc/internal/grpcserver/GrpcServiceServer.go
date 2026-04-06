package grpcserver

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
	"github.com/hiveot/hivekit/go/modules/transports/grpc/internal"

	// grpcapi "github.com/hiveot/hivekit/go/modules/transports/grpc/api"
	"github.com/hiveot/hivekit/go/msg"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/encoding"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// used by gRPC for API check
type IGrpcServiceServer interface {
	NewMsgStream(msgType string, grpcStream grpc.ServerStream) error
	Ping(ctx context.Context, input string) (string, error)
	// mustEmbedUnimplementedGrpcServiceServer()
}

// ServiceDesc for server side registration of the grpc methods and streams
// var ServiceDesc = grpc.ServiceDesc{
// 	// ServiceName: "GrpcServiceServer2",
// 	ServiceName: "pleasesetaservicename",
// 	HandlerType: (*IGrpcServiceServer)(nil),
// 	Methods: []grpc.MethodDesc{
// 		{
// 			MethodName: "ping",
// 			Handler: func(srv interface{}, ctx context.Context,
// 				dec func(any) error,
// 				interceptor grpc.UnaryServerInterceptor) (any, error) {

// 				var input string
// 				err := dec(&input)
// 				if err == nil {
// 					return srv.(*GrpcServiceServer).Ping(ctx, input)
// 				}
// 				return input, err
// 			},
// 		},
// 	},
// 	Streams: []grpc.StreamDesc{
// 		// notification stream
// 		{
// 			StreamName: grpcapi.StreamNameNotification,
// 			// this handler serves the stream with StreamName name.
// 			Handler: func(srv interface{}, stream grpc.ServerStream) error {
// 				return srv.(*GrpcServiceServer).NewMsgStream(msg.MessageTypeNotification, stream)
// 			},
// 			ServerStreams: true,
// 			ClientStreams: true,
// 		},
// 		// request/response stream
// 		{
// 			StreamName: grpcapi.StreamNameRequestResponse,
// 			// this handler serves the stream with StreamName name.
// 			Handler: func(srv interface{}, stream grpc.ServerStream) error {
// 				return srv.(*GrpcServiceServer).NewMsgStream(msg.MessageTypeRequest, stream)
// 			},
// 			ServerStreams: true,
// 			ClientStreams: true,
// 		},
// 	},
// 	Metadata: "grpc_transport.proto",
// }

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

// Add a new stream to listen on
func (srv *GrpcServiceServer) AddStream(
	name string, handler func(clientID, cid string, grpcStream grpc.ServerStream) error) error {

	srv.serviceDesc.Streams = append(srv.serviceDesc.Streams, grpc.StreamDesc{
		StreamName: name,
		// this handler serves the stream with StreamName name.
		Handler: func(srvApi interface{}, stream grpc.ServerStream) error {
			srv := srvApi.(*GrpcServiceServer)
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

// NewMsgStream is the handler of an incoming messaging stream connection
// This extracts the clientID and ConnectionID metadata from the stream and
// invokes the registered stream server.
// The stream closes when the serve handler returns.
//
// name is the name of the stream: eg "notification"
// Returning from serveHandler closes the stream.
func (srv *GrpcServiceServer) NewMsgStream(name string, grpcStream grpc.ServerStream) error {
	return fmt.Errorf("no longer valid. Use AddStream instead")
	// clientID, cid, err := srv.GetRequestParams(grpcStream.Context())
	//
	//	if err != nil {
	//		return err
	//	}
	//
	// slog.Info("MsgStream: Service received a new stream", "clientID", clientID, "cid", cid, "msgType", name)
	// // serveHandler should block until the stream is closed
	// // FIXME: generalize - lookup available stream handlers
	//
	//	if name == msg.MessageTypeNotification {
	//		err = srv.serveNotifStream(clientID, cid, grpcStream)
	//	} else {
	//
	//		err = srv.serveReqRespStream(clientID, cid, grpcStream)
	//	}
	//
	// return err
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
	grpcServer.RegisterService(&srv.serviceDesc, srv)

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
}

// Start the GRPC server and listen for incoming connections.
// Note that the proto file only defines a single bi-directional stream so all
// traffic goes over these streams.
//
// serveHandler is called when a stream connection is established and ready to be served.
// This should block until the connection closes.
//
// Use NewGrpcServiceStream(grpcStream) to create a buffered concurrently safe
// connection for this stream.
//
//	lis is the network to listen on
//	tlsCert is the TLS certificate to use for secure connections, or nil for insecure
//	serviceName is the service name the streams are reachable under
//	serveHandler is called with the raw stream when one is opened by a client
//	grpcAuthn is the grpc connection authenticator
//	respTimeout is the messaging timeout
func NewGrpcServiceServer(lis net.Listener,
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
	encoding.RegisterCodec(internal.JsonCodec{})

	var _ IGrpcServiceServer = srv // interface check

	// create a service description with a default ping method
	// streams can be added with 'AddStream' before Start is called.
	srv.serviceDesc = grpc.ServiceDesc{
		ServiceName: serviceName,
		HandlerType: (*IGrpcServiceServer)(nil),
		Methods: []grpc.MethodDesc{
			{
				MethodName: "ping",
				Handler: func(srv interface{}, ctx context.Context,
					dec func(any) error,
					interceptor grpc.UnaryServerInterceptor) (any, error) {

					var input string
					err := dec(&input)
					if err == nil {
						return srv.(*GrpcServiceServer).Ping(ctx, input)
					}
					return input, err
				},
			},
		},
	}
	return srv
}

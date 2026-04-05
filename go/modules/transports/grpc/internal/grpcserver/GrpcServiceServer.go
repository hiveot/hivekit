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
	grpcapi "github.com/hiveot/hivekit/go/modules/transports/grpc/api"
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

// tbd: might not be needed
type IGrpcServiceServer2 interface {
	MsgStream(grpcStream grpc.ServerStream) error
	Ping(ctx context.Context) (string, error)
	// mustEmbedUnimplementedGrpcServiceServer()
}

func pingHandler(srv any, ctx context.Context,
	dec func(any) error,
	interceptor grpc.UnaryServerInterceptor) (any, error) {

	out, err := srv.(IGrpcServiceServer2).Ping(ctx)
	res := []byte(out)

	return res, err
}

// ServiceDesc for server side registration of this
var ServiceDesc = grpc.ServiceDesc{
	// ServiceName: "GrpcServiceServer2",
	ServiceName: "grpcapi.GrpcService",
	HandlerType: (*IGrpcServiceServer2)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "ping",
			Handler:    pingHandler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			// a not very original name for this stream
			StreamName: grpcapi.GrpcTransportStreamName,
			// this handler serves the stream with StreamName name.
			Handler: func(srv interface{}, stream grpc.ServerStream) error {
				return srv.(*GrpcServiceServer).MsgStream(stream)
			},
			ServerStreams: true,
			ClientStreams: true,
		},
	},
	Metadata: "grpc_transport.proto",
}

// GRPC server handler of protobuf defined methods.
// This currently only implements the Ping and MsgStream methods.
// The future goal is to remove protobuf dependency entirely and just use grpc.
type GrpcServiceServer struct {
	grpcAuthn *GrpcAuthenticator

	// the underlying GRPC server
	grpcServer *grpc.Server

	// callback for serving a new stream
	serveHandler func(clientID string, cid string, grpcStream grpc.ServerStream) error

	// how long to wait for a response after sending a request
	respTimeout time.Duration

	// todo
	rnrChan *msg.RnRChan
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

// Handler an incoming connection for a MsgStream.
// This extracts the clientID and ConnectionID metadata from the stream and
// invokes the registered stream server.
// The stream closes when the serve handler returns.
//
// Returning from serveHandler closes the stream.
func (srv *GrpcServiceServer) MsgStream(grpcStream grpc.ServerStream) error {

	clientID, cid, err := srv.GetRequestParams(grpcStream.Context())
	if err != nil {
		return err
	}
	slog.Info("MsgStream: Service received stream connection", "clientID", clientID, "cid", cid)
	// serveHandler should block until the stream is closed
	err = srv.serveHandler(clientID, cid, grpcStream)
	return err
}

// Handler of ping message returns pong
func (srv *GrpcServiceServer) Ping(ctx context.Context) (result string, err error) {
	clientID, cid, err := srv.GetRequestParams(ctx)
	_ = err
	log.Printf("Ping: ping received from clientID '%s', cid='%s'", clientID, cid)
	return "pong", nil
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
//	serveHandler is called with the raw stream when one is opened by a client
//	grpcAuthn is the grpc connection authenticator
//	respTimeout is the messaging timeout
func StartGrpcServiceServer(lis net.Listener,
	tlsCert *tls.Certificate,
	serveHandler func(clientID string, cid string, grpcStream grpc.ServerStream) error,
	grpcAuthn *GrpcAuthenticator,
	respTimeout time.Duration,
) (*GrpcServiceServer, error) {

	srv := &GrpcServiceServer{
		grpcAuthn:    grpcAuthn,
		respTimeout:  respTimeout,
		serveHandler: serveHandler,
	}

	var grpcServer *grpc.Server
	var opts = make([]grpc.ServerOption, 0)

	// Create the TLS credentials
	// creds, err := credentials.NewServerTLSFromFile(certFile, keyFile)
	if tlsCert != nil {
		creds := credentials.NewServerTLSFromCert(tlsCert)
		// Create an array of gRPC options with the credentials
		opts = append(opts, grpc.Creds(creds))
	}

	// not sure raw codec needs to do something for RRN messages or is []byte passthrough sufficient?
	// !The incoming request content-type header must match the codec name.
	// or force it using grpc.ForceServerCodec()
	// note: registration applies to all client and servers
	encoding.RegisterCodec(internal.RawCodec{})

	// auth and stuff
	opts = append(opts, grpc.UnaryInterceptor(srv.unaryInterceptor))
	opts = append(opts, grpc.StreamInterceptor(srv.streamInterceptor))
	// opts = append(opts, encoding.Codec(internal.RawCodec{}))

	grpcServer = grpc.NewServer(opts...)

	var _ IGrpcServiceServer2 = srv // interface check

	// register the callback handler for incoming streams
	grpcServer.RegisterService(&ServiceDesc, srv)

	// grpcapi.RegisterGrpcServiceServer(grpcServer, srv)
	srv.grpcServer = grpcServer

	//---

	// grpcServer.Hand

	//---

	// start the server
	slog.Info("StartGrpcServer: starting  gRPC server", slog.String("Address", lis.Addr().String()))
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			slog.Error("StartGrpcServer: failed to serve", "err", err.Error())
		}
	}()
	time.Sleep(time.Millisecond)
	return srv, nil
}

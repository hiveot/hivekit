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
	"github.com/hiveot/hivekit/go/msg"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// GRPC server handler of protobuf defined methods.
// This currently only implements the Ping and MsgStream methods.
// The future goal is to remove protobuf dependency entirely and just use grpc.
type GrpcServiceServer struct {
	grpcapi.UnimplementedGrpcServiceServer

	grpcAuthn *GrpcAuthenticator

	// the underlying GRPC server
	grpcServer *grpc.Server

	// callback for connection status changes
	connectHandler func(strm *GrpcServiceStream) error

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
			return status.Errorf(codes.Unauthenticated, "Unauthenticated: %w", err)
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
	clientID string, cid string, validUntil time.Time, err error) {

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		slog.Error("GetRequestParams: missing auth metadata context")
		return "", "", validUntil, fmt.Errorf("missing metadata")
	}
	clientID = strings.Join(md[transports.ClientIDContextID], "")
	cid = strings.Join(md[transports.ClientCIDContextID], "")
	return clientID, cid, validUntil, err
}

// Handler an incoming connection for a MsgStream.
// MsgStream is defined in protobuf.
func (srv *GrpcServiceServer) MsgStream(grpcStream grpcapi.GrpcService_MsgStreamServer) error {

	clientID, cid, validUntil, err := srv.GetRequestParams(grpcStream.Context())
	_ = validUntil
	if err != nil {
		return err
	}
	slog.Info("MsgStream: Service received stream connection", "clientID", clientID, "cid", cid)
	strm := NewGrpcServiceStream(grpcStream, srv.respTimeout)

	srv.connectHandler(strm)

	return nil
}

// Handler of ping message
func (srv *GrpcServiceServer) Ping(ctx context.Context, in *grpcapi.PingMsg) (*grpcapi.PingMsg, error) {
	clientID, cid, _, err := srv.GetRequestParams(ctx)
	_ = err
	log.Printf("Ping: ping received from clientID '%s', cid='%s'", clientID, cid)
	return &grpcapi.PingMsg{Text: in.Text}, nil
}

// graceful stop of the server
func (srv *GrpcServiceServer) Stop() {
	if srv.grpcServer != nil {
		srv.grpcServer.GracefulStop()
		srv.grpcServer = nil
	}
}

// Start the GRPC server.
//
// The connectHandler is called when a connection is established that needs authentication
// this can be a new message stream or a unary request call.
//
//	lis is the network to listen on
//	tlsCert is the TLS certificate to use for secure connections, or nil for insecure
//	connectHandler is called when a messaging stream is opened by a client
//	respTimeout is the messaging timeout
func StartGrpcServiceServer(lis net.Listener,
	tlsCert *tls.Certificate,
	connectHandler func(strm *GrpcServiceStream) error,
	respTimeout time.Duration,
) (*GrpcServiceServer, error) {

	srv := &GrpcServiceServer{
		respTimeout:    respTimeout,
		connectHandler: connectHandler,
		// grpcStream: nil,
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
	// auth and stuff
	opts = append(opts, grpc.UnaryInterceptor(srv.unaryInterceptor))
	opts = append(opts, grpc.StreamInterceptor(srv.streamInterceptor))

	// creds, err := credentials.NewClientTLSFromFile("cert/server.crt", "")
	// dialOpt := grpc.WithTransportCredentials(insecure.NewCredentials())
	// opts = append(opts, dialOpt)

	grpcServer = grpc.NewServer(opts...)
	grpcapi.RegisterGrpcServiceServer(grpcServer, srv)
	srv.grpcServer = grpcServer

	// start the server
	log.Printf("StartGrpcServer: starting  gRPC server on %s", lis.Addr().String())
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			slog.Error("StartGrpcServer: failed to serve", "err", err.Error())
		}
	}()
	return srv, nil
}

package grpcapi

import (
	"github.com/hiveot/hivekit/go/modules/transports"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// constants

const (
	// Hiveot gRPC module ID
	HiveotGrpcModuleID = "hiveot-grpc"

	GrpcTransportServiceName = "grpcTransport"
	// the stream names used in client and server
	StreamNameNotification    = "notification"
	StreamNameRequestResponse = "requestresponse"
)

// The default socket path for the grpc UDS server
// var HiveotGrpcSocketPath = filepath.Join(os.TempDir(), "/hiveot/grpc.sock")
// var HiveotGrpcSocketPath = "/tmp/hiveot/grpc.sock"

// error result codes
var ErrMissingMetadata = status.Errorf(codes.InvalidArgument, "missing metadata")
var ErrInvalidToken = status.Errorf(codes.PermissionDenied, "invalid token")
var ErrConnectionClosed = status.Errorf(codes.Canceled, "connection is closed")
var ErrClientTooSlow = status.Errorf(codes.ResourceExhausted, "client is too slow to receive messages")

// Interface of the Hiveot gRPC transport server module
type IGrpcTransportServer interface {
	transports.ITransportServer

	// todo: future API  for servicing the module
}

// API for use by all client/server streaming endpoints
type IMsgStream interface {
	SendMsg(msg any) error
	RecvMsg(dest any) error
}

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
)

// The default socket path for the grpc UDS server
// var HiveotGrpcSocketPath = filepath.Join(os.TempDir(), "/hiveot/grpc.sock")
// var HiveotGrpcSocketPath = "/tmp/hiveot/grpc.sock"

// error result codes
var ErrMissingMetadata = status.Errorf(codes.InvalidArgument, "missing metadata")
var ErrInvalidToken = status.Errorf(codes.PermissionDenied, "invalid token")
var ErrConnectionClosed = status.Errorf(codes.Canceled, "connection is closed")
var ErrClientTooSlow = status.Errorf(codes.ResourceExhausted, "client is too slow to receive messages")

// interface for the protobuf message stream. Used to 'equalize' the client
// and server stream interfaces for the buffered stream implementation.
type IGrpcMessageStream interface {
	Send(*GrpcMsg) error
	Recv() (*GrpcMsg, error)
}

// Interface of the Hiveot gRPC transport server module
type IGrpcTransportServer interface {
	transports.ITransportServer

	// todo: future API  for servicing the module
}

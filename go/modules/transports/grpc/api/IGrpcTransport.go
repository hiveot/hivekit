package grpcapi

import (
	"github.com/hiveot/hivekit/go/modules/transports"
)

// constants

const (
	// Hiveot gRPC module ID
	HiveotGrpcModuleType = "hiveot-grpc"

	// there is no WoT gRPC specification

	// The grpc service that identifies the streams
	GrpcTransportServiceName = "grpcTransport"
	// the stream names used in client and server
	StreamNameNotification    = "notification"
	StreamNameRequestResponse = "requestresponse"
)

// The default socket path for the grpc UDS server
// var HiveotGrpcSocketPath = filepath.Join(os.TempDir(), "/hiveot/grpc.sock")
// var HiveotGrpcSocketPath = "/tmp/hiveot/grpc.sock"

// Interface of the Hiveot gRPC transport server module
type IGrpcTransportServer interface {
	transports.ITransportServer

	// todo: future API  for servicing the module
}

package grpctransport

import (
	"github.com/hiveot/hivekit/go/modules/transport"
)

// constants

const (
	// Hiveot gRPC module IDs
	HiveotGrpcClientModuleType = "hiveot-grpc-client"
	HiveotGrpcServerModuleType = "hiveot-grpc-server"

	// there is no WoT gRPC specification

	// The default gRPC server listening URL
	DefaultGrpcURL = "unix:///var/hiveot/hivekit.sock"

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
	transport.ITransportServer

	// todo: future API  for servicing the module
}

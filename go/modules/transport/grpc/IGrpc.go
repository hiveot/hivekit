package grpc

import (
	"github.com/hiveot/hivekit/go/api"
)

// constants

const (
	// Hiveot gRPC module IDs
	HiveotGrpcClientModuleType = "hiveot-grpc-client"
	HiveotGrpcServerModuleType = "hiveot-grpc-server"

	// there is no WoT gRPC specification

	// The default gRPC server listening URL
	DefaultGrpcUnixURL = "unix:///tmp/hiveot/grpc-server.sock"
	DefaultGrpcTcpURL  = "tcp://localhost:50051/hiveot/grpc"

	// The grpc service that identifies the streams
	GrpcTransportServiceName = "grpcTransport"

	// the stream names used in client and server
	StreamNameNotification    = "notification"
	StreamNameRequestResponse = "requestresponse"
)

// The default socket path for the grpc UDS server
// var HiveotGrpcSocketPath = filepath.Join(os.TempDir(), "/hiveot/grpc.sock")
// var HiveotGrpcSocketPath = "/tmp/hiveot/grpc.sock"

// optional configuration to include factory ModuleDefinition.Config
type GrpcConfig struct {
	// gRPC server listening URL: DefaultGrpcUnixURL or DefaultGrpcTcpURL
	URL string
}

// Interface of the Hiveot gRPC transport server module
type IGrpcTransportServer interface {
	api.ITransportServer

	// todo: future API  for servicing the module
}

package grpc

import (
	"time"

	grpcapi "github.com/hiveot/hivekit/go/modules/transports/grpc/api"
	"github.com/hiveot/hivekit/go/modules/transports/grpc/internal/grpcserver"
)

// NewHiveotGrpcServer creates a hiveot gRPC transport server.
//
// # This uses the HiveOT RRN messages as the payload.
//
// socketPath is the UDS path to listen on
// respTimeout is the time the server waits for a response when receiving requests. defaults to 3sec
//
// Use SetRequestSink to set the handler for requests send by consumers
// Use SetNotificationSink to set the handler for notifications send by agents.
func NewHiveotGrpcServer(socketPath string, respTimeout time.Duration) grpcapi.IGrpcTransportServer {
	return grpcserver.NewHiveotGrpcUDSServer(socketPath, respTimeout)
}

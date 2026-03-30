package grpc

import (
	"time"

	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/modules/transports/grpc/internal/grpcclient"
)

// NewHiveotGrpcClient creates a hiveot gRPC transport client.
//
// This uses the HiveOT RRN messages as the payload.
//
// socketPath is the UDS path to connect to
// respTimeout is the time the client waits for a response when receiving requests. defaults to 3sec
//
// Use SetRequestSink to set the handler for requests send by consumers
// Use SetNotificationSink to set the handler for notifications send by agents.
func NewHiveotGrpcClient(socketPath string, respTimeout time.Duration) transports.ITransportClient {
	return grpcclient.NewHiveotGrpcClient(socketPath, respTimeout)
}

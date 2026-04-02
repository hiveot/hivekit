package grpc

import (
	"crypto/tls"
	"time"

	grpcapi "github.com/hiveot/hivekit/go/modules/transports/grpc/api"
	"github.com/hiveot/hivekit/go/modules/transports/grpc/internal/grpcserver"
)

// NewHiveotGrpcServer creates a hiveot gRPC transport server.
//
// # This uses the HiveOT RRN messages as the payload.
//
// connectURL is the URL to listen on, e.g. unix:///tmp/hivekit.sock or tcp://localhost:50051
// tlsCert is the TLS certificate to use for secure connections, or nil for insecure
// respTimeout is the time the server waits for a response when receiving requests. defaults to 3sec
//
// Use SetRequestSink to set the handler for requests send by consumers
// Use SetNotificationSink to set the handler for notifications send by agents.
func NewHiveotGrpcServer(connectURL string, tlsCert *tls.Certificate, respTimeout time.Duration) grpcapi.IGrpcTransportServer {
	return grpcserver.NewHiveotGrpcServer(connectURL, tlsCert, respTimeout)
}

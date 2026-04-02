package grpc

import (
	"crypto/x509"

	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/modules/transports/grpc/internal/grpcclient"
)

// NewHiveotGrpcClient creates a hiveot gRPC transport client.
//
// This uses the HiveOT RRN messages as the payload.
//
// addr is the UDS path to connect to
// caCert of the CA used by the server in establishing transport credentials
//
// Use SetTimeout to change the default response timeout
// Use SetRequestSink to set the handler for requests send by consumers
// Use SetNotificationSink to set the handler for notifications send by agents.
func NewHiveotGrpcClient(addr string, caCert *x509.Certificate) transports.ITransportClient {
	return grpcclient.NewGrpcTransportClient(addr, caCert)
}

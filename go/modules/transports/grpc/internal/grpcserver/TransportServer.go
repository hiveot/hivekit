package grpcserver

import (
	"time"

	"github.com/hiveot/hivekit/go/modules/transports"
	"github.com/hiveot/hivekit/go/msg"
	"google.golang.org/grpc"
)

const DefaultUDSModuleID = "hiveotuds"

type GrpcTransportServer struct {
	transports.TransportServerBase

	grpcConn *GrpcServiceServer

	udsPath    string
	grpcServer *grpc.Server

	respTimeout time.Duration
}

// GetProtocolType returns type identifier of the server protocol as defined by its module
func (m *GrpcTransportServer) GetProtocolType() string {
	return transports.HiveotUdsProtocolType
}

// Handle a notification this module (or downstream in the chain) subscribed to.
// Notifications are forwarded to their upstream sink, which for a server is the
// client.
func (m *GrpcTransportServer) HandleNotification(notif *msg.NotificationMessage) {
	m.SendNotification(notif)
}

func (m *GrpcTransportServer) HandleRequest(
	req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {
	return nil
}

func (m *GrpcTransportServer) Start(yamlConfig string) (err error) {
	// FIXME: use the URL scheme to support network tcp and unix sockets
	// connectURL := fmt.Sprintf("unix://%s", m.udsPath)
	m.Init(DefaultUDSModuleID, "", m.udsPath, nil)

	// start listening on unix sockets
	// lis, err := net.Listen("unix", m.udsPath)
	if err != nil {
		return err
	}
	// grpcServer := grpc.NewServer()
	// grpcConnection := StartServeGrpc(m.ForwardRequest, m.ForwardNotification, m.respTimeout)
	// grpcapi.RegisterGrpcTransportServer(grpcServer, grpcConnection)
	// go func() {
	// 	err = m.grpcServer.Serve(lis)
	// }()
	return nil
}

// Stop any running actions
func (m *GrpcTransportServer) Stop() {
}

// GRPC server using UDS
func NewHiveotGrpcUDSServer(udsPath string, respTimeout time.Duration) *GrpcTransportServer {
	srv := &GrpcTransportServer{
		udsPath:     udsPath,
		respTimeout: respTimeout,
	}
	return srv
}

// GRPC server using HTTP
func NewHiveotGrpcHttpServer(httpServer transports.IHttpServer, respTimeout time.Duration) *GrpcTransportServer {

	srv := &GrpcTransportServer{
		// httpServer:  httpServer,
		respTimeout: respTimeout,
	}
	return srv
}

package grpcserver

import (
	"crypto/tls"
	"fmt"
	"net"
	"time"

	"github.com/hiveot/hivekit/go/modules/transports"
	grpcapi "github.com/hiveot/hivekit/go/modules/transports/grpc/api"
	"github.com/hiveot/hivekit/go/msg"
)

const DefaultUDSModuleID = "hiveotuds"

// GrpcTransportServer is the transport server using gRPC connections.
//
// This implements both ITransportServer and IHiveModule interfaces.
// The embedded TransportServerBase is used for managing connections and forwarding messages to sinks.
type GrpcTransportServer struct {
	transports.TransportServerBase

	tlsCert *tls.Certificate

	grpcService *GrpcServiceServer

	connectURL string
	// grpcServer *grpc.Server

	respTimeout time.Duration
}

// GetProtocolType returns type identifier of the server protocol as defined by its module
func (m *GrpcTransportServer) GetProtocolType() string {
	return transports.HiveotGrpcProtocolType
}

// a request passed to this server is forwarded to the connection with the matching ID
// this is intended for passing requests to agents that have a reverse connection.
func (m *GrpcTransportServer) HandleRequest(
	req *msg.RequestMessage, replyTo msg.ResponseHandler) (err error) {

	// first attempt to procss the when targeted at this module
	if req.ThingID == m.GetModuleID() {
		// currently nothing to do here
		// err = m.msgAPI.HandleRequest(req, replyTo)
		err = fmt.Errorf("HandleRequest: no operations for this module '%s' re defined: ", m.GetModuleID())
	} else {
		err = m.TransportServerBase.HandleRequest(req, replyTo)
	}
	return err
}

// The grpc service callback handler for incoming stream connections.
// This creates a new transport connection for the stream and blocks until the stream is closed.
func (m *GrpcTransportServer) serveStream(clientID string, cid string, grpcStream grpcapi.GrpcService_MsgStreamServer) error {

	// Create a hiveot transport connection for this stream.
	c := StartGrpcTransportConnection(clientID, cid, grpcStream,
		m.ForwardRequest, m.ForwardNotification, m.respTimeout)

	m.AddConnection(c)
	// must block until connection closes
	c.WaitUntilDisconnect()
	m.RemoveConnection(c)
	return nil
}

func (m *GrpcTransportServer) Start(yamlConfig string) (err error) {
	// FIXME: use the URL scheme to support network tcp and unix sockets
	// connectURL := fmt.Sprintf("unix://%s", m.udsPath)
	m.Init(DefaultUDSModuleID, "hiveot-uds", m.connectURL, nil)

	// start listening on unix sockets
	lis, err := net.Listen("unix", m.connectURL)
	if err != nil {
		return err
	}

	m.grpcService, err = StartGrpcServiceServer(lis, nil, m.serveStream, time.Minute)
	if err != nil {
		lis.Close()
		return err
	}

	return err
}

// Stop any running actions
func (m *GrpcTransportServer) Stop() {
}

// GRPC server using UDS or TCP sockets
//
// connectURL is the URL to listen on, e.g. unix://{/path.sock} or tcp://localhost:{port}
// tlsCert is the TLS certificate to use for secure connections, or nil for insecure
// respTimeout is the time the server waits for a response when sending requests. defaults to 3sec
func NewHiveotGrpcServer(connectURL string, tlsCert *tls.Certificate, respTimeout time.Duration) *GrpcTransportServer {

	srv := &GrpcTransportServer{
		connectURL:  connectURL,
		tlsCert:     tlsCert,
		respTimeout: respTimeout,
	}
	return srv
}

// GRPC server using embedded HTTP server
// httpServer is the HTTP server to use for handling incoming connections. The gRPC server will be
// mounted on a specific path, e.g. /hiveot/grpc.
// func NewHiveotGrpcHttpServer(httpServer transports.IHttpServer, respTimeout time.Duration) *GrpcTransportServer {

// 	srv := &GrpcTransportServer{
// 		httpServer:  httpServer,
// 		respTimeout: respTimeout,
// 	}
// 	return srv
// }

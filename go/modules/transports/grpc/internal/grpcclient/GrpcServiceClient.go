package grpcclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/hiveot/hivekit/go/modules/transports"
	grpcapi "github.com/hiveot/hivekit/go/modules/transports/grpc/api"
	"github.com/teris-io/shortid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
)

const ClientMsgChanSize = 30

// Grpc messaging client.
// This uses the BufferedStream for sending and receiving messages on the GRPC stream
// connection. The buffer size is currently fixed to 30 messages, which should be
// sufficient for most use cases.
//
// This also implements the PerTransportBundle PerRPCCredentials interface
type GrpcServiceClient struct {

	// buffered stream wrapper around the protobuf stream.
	bufStream *BufferedStream

	// caCert in case the connectURL is an ip connection address
	caCert *x509.Certificate

	// clientID and connectionID to include in the grpc metadata.
	clientID     string
	connectionID string

	// URL to connect to. see also https://github.com/grpc/grpc/blob/master/doc/naming.md
	// unix:///path/to/socket
	// ipv4://address:[port][,address[:port]]
	// dns://address:[port]
	connectURL string

	// conn         net.Conn
	grpcConn          *grpc.ClientConn
	grpcServiceClient grpcapi.GrpcServiceClient // interface
	//
	// msgStream       grpcapi.GrpcService_MsgStreamClient // interface
	msgStreamCancel func()
	// ping from the gRPC protobuf definition
	pingHandler func(context.Context, any) (reply string, err error)

	// callback for incoming messages
	recvHandler func(msgType string, rawJson string)

	// mutex for controlling writing and closing
	mux sync.RWMutex

	respTimeout time.Duration
}

// Close disconnects
func (cl *GrpcServiceClient) Close() {
	cl.mux.Lock()
	defer cl.mux.Unlock()
	if cl.bufStream != nil {
		cl.bufStream.Close()
		cl.bufStream = nil
	}
	if cl.msgStreamCancel != nil {
		cl.msgStreamCancel()
		cl.msgStreamCancel = nil
	}
	if cl.grpcConn != nil {
		cl.grpcConn.Close()
		cl.grpcConn = nil
	}
	slog.Info("client Close: ending read/write loops")
}

// Initiate a connection to the grpc server
func (cl *GrpcServiceClient) Connect() (err error) {
	cl.mux.Lock()
	defer cl.mux.Unlock()

	// fail if a connection already exists
	if cl.grpcConn != nil {
		return fmt.Errorf("Connect: A connection already exists. Close first.")
	}

	dialOpts := []grpc.DialOption{}

	// If a CA certificate is set the use Transport credentials
	if cl.caCert != nil {
		// configure the TransportCredentials dial option
		var clientCertList []tls.Certificate
		// TODO: support client cert auth

		caCertPool := x509.NewCertPool()
		caCertPool.AddCert(cl.caCert)

		tlsConfig := &tls.Config{
			RootCAs:            caCertPool,
			InsecureSkipVerify: false,
			Certificates:       clientCertList,
		}
		tlsCreds := credentials.NewTLS(tlsConfig)
		tlsCredOpt := grpc.WithTransportCredentials(tlsCreds)
		dialOpts = append(dialOpts, tlsCredOpt)
	} else {
		nocreds := grpc.WithTransportCredentials(insecure.NewCredentials())
		dialOpts = append(dialOpts, nocreds)
	}
	// this client implements the PerRPCCredentials interface
	// see: GetRequestMetadata and RequireTransportSecurity
	// rpcCredOpt := grpc.WithPerRPCCredentials(cl)
	// dialOpts = append(dialOpts, rpcCredOpt)

	cl.grpcConn, err = grpc.NewClient(cl.connectURL, dialOpts...)
	if err != nil {
		slog.Error("Connect: NewClient failed", "err", err.Error())
		return err
	}
	grpcServiceClient := grpcapi.NewGrpcServiceClient(cl.grpcConn)

	ctx, cancelFn := context.WithCancel(context.Background())
	cl.msgStreamCancel = cancelFn
	msgStream, err := grpcServiceClient.MsgStream(ctx)
	if err != nil {
		slog.Error("Connect: MsgStream failed", "err", err.Error())
		return err
	}

	// use buffered stream for sending and receiving
	cl.bufStream = NewGrpcBufferedStream(msgStream, cl.recvHandler, cl.respTimeout)
	cl.grpcServiceClient = grpcServiceClient

	return nil
}

// // GetConnectionID returns the client's connection details
func (cl *GrpcServiceClient) GetConnectionID() string {
	return cl.connectionID
}

func (cl *GrpcServiceClient) IsConnected() bool {
	return cl.bufStream.IsConnected()
}

func (cl *GrpcServiceClient) Ping(pingText string) (reply string, err error) {
	ctx, cancelFn := context.WithTimeout(context.Background(), cl.respTimeout)
	defer cancelFn()
	replyMsg, err := cl.grpcServiceClient.Ping(ctx, &emptypb.Empty{})
	if err != nil {
		return "", err
	}
	return replyMsg.Text, nil
}

// PerRPCCredentials:GetRequestMetadata
func (cl *GrpcServiceClient) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	return map[string]string{
		transports.ClientIDContextID:  cl.clientID,
		transports.ClientCIDContextID: cl.connectionID,
	}, nil
}

// PerRPCCredentials:RequireTransportSecurity
func (cl *GrpcServiceClient) RequireTransportSecurity() bool { return true }

// Send a message to the server
func (cl *GrpcServiceClient) Send(msgType string, jsonPayload string) {
	cl.bufStream.Send(msgType, jsonPayload)
}

// WaitUntilDisconnect waits until the client connection is closed.
func (cl *GrpcServiceClient) WaitUntilDisconnect() {
	cl.bufStream.WaitUntilDisconnect()
}

// Create a client for the GRPC protocol
// caCert is optional for use with tcp sockets
func NewGrpcServiceClient(clientID string, connectURI string, caCert *x509.Certificate, respTimeout time.Duration,
	msgHandler func(msgType string, jsonRaw string),
) *GrpcServiceClient {

	cl := &GrpcServiceClient{
		caCert:       caCert,
		clientID:     clientID,
		connectionID: shortid.MustGenerate(),
		connectURL:   connectURI,
		recvHandler:  msgHandler,
		respTimeout:  respTimeout,
	}
	return cl
}

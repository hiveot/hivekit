package internal

import (
	"fmt"
)

// raw codec and service descriptor
type RawCodec struct {
}

func (c RawCodec) Name() string {
	return "rawcodec"
}

func (RawCodec) Marshal(v interface{}) ([]byte, error) {
	if data, ok := v.([]byte); ok {
		return data, nil
	}
	return nil, fmt.Errorf("expected []byte")
}

func (RawCodec) Unmarshal(data []byte, v interface{}) error {
	if out, ok := v.(*[]byte); ok {
		*out = data
		return nil
	}
	return fmt.Errorf("expected *[]byte")
}

func (RawCodec) String() string {
	return "raw"
}

// server side stream handler creation
// not sure why the client needs this though as it gets a server stream?
//
// srv is the server that implements the handler method of type HandlerType
// stream is the grpc stream - server?
// func _GrpcService_MsgStream_Handler(
// 	srv interface{}, stream grpc.ServerStream) error {
// 	// this returns the server stream implementation giving the server stream
// 	return srv.(GrpcServiceServer).MsgStream(&grpcServiceMsgStreamServer{stream})
// }

// // HiveotServiceDesc for service registration
// var HiveotServiceDesc = grpc.ServiceDesc{
// 	ServiceName: "grpcapi.GrpcService",
// 	HandlerType: (*GrpcServiceServer)(nil),
// 	Methods:     []grpc.MethodDesc{
// 		// {
// 		// 	MethodName: "ping",
// 		// 	Handler:    _GrpcService_Ping_Handler,
// 		// },
// 	},
// 	Streams: []grpc.StreamDesc{
// 		{
// 			StreamName:    "MsgStream",
// 			Handler:       _GrpcService_MsgStream_Handler,
// 			ServerStreams: true,
// 			ClientStreams: true,
// 		},
// 	},
// 	Metadata: "grpc_transport.proto",
// }

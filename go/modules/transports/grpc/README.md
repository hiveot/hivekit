# grpc - HiveOT gRPC Transport

'[grpc](https://grpc.io/)' is a high performance Remote Procedure Call framework that can run in any environment.

This transport module passes RRN messages as-is between client and server using a bi-directional gRPC stream.

## Status

This transport module is functional but breaking changes can be expected.

TODO-1: Support for tcp sockets and URL
TODO-3: use a separate notification stream
TODO-5: test CA cert when using tcp sockets
TODO-6: authenticate with Client TLS cert when using tcp sockets

## Summary

The primary purpose of the gRPC transport is to support Unix Domain Sockets for high performance (10K msg/sec on rPi3+, 100K/sec on i5) local inter-process communication. Tcp sockets are also supported, expanding its use to the network.

It is mainly intended for agents that use reverse connections Since they represent multiple devices or services, performance is more important than with a single isolated device.

## Configuration

The module configuration consists of:

1. The connection URL which indicates the scheme (unix or tcp) and the socket or tcp address.
2. A TLS certificate on the server and CA certificate on the client.
3. An authenticator for validating client credentials. This is optional when using UDS.
4. Optional timeout to facilitate testing and debugging. The default is defined in DefaultRpcTimeout.

## grpclib

The lib directory contains a gRPC wrapper for creating bi-directional streams on the fly without protobuf. The user can create a stream on the server using 'CreateStream(name, handler)' and connect to it with the client using 'ConnectStream(name, handler)'. The test case shows a full example.

Also included is a stream buffer that adds these features:

- accept concurrent sending of messages (grpc stream send is not concurrent)
- add a send buffer for immediate return after calling Send
- flow control. Dynamically delay the caller time when the send buffer is 50% full and slowly reduce the delay when the buffer empties out.
- if the send buffer is full then add an extra 10usec delay and repeat up to 10 times (configurable). After that returns ErrClientTooSlow error.

No need for grpc magic and types and stuff. Simply add the stream on the server and connect to it on the client.
This registers the 'jsonCodec' in case of complex objects. Or the user can simply send and receive string or []byte arrays and use their own encoder.

On an intel i5 4570S, 2.9GHz this transfers 350K 300byte messages/sec or 270K 1K messages/sec using unix sockets.

Server gist:

```go
	lis, err := net.Listen("unix", "/run/myapp.sock")
	grpcServer := grpclib.NewGrpcServiceServer(
       lis, nil, "myservicename", grpcAuthn, time.Minute)
	grpcServer.CreateStream("stream-1", serverStreamHandler)
   	err = m.grpcService.Start()
```

Client gist:

```go
	grpcClient = grpclib.NewGrpcServiceClient(
        connectURL, nil, time.Minute, "myservicename", clientStreamHandler)
	err := grpcClient.ConnectWithToken(clientID, token)
	_, err = grpcClient.Ping("")
	strm, err = grpcClient.ConnectStream("stream-1")
    // this blocks
	strm.WaitUntilDisconnect(name)
```

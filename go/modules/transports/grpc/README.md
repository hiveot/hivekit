# grpc - HiveOT gRPC Transport

'[grpc](https://grpc.io/)' is a high performance Remote Procedure Call framework that can run in any environment.

This transport module passes RRN messages as-is between client and server using a bi-directional gRPC stream.

## Status

This transport module is functional but breaking changes can be expected.

TODO-3: use a separate notification stream
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

- accept concurrent sending of messages (grpc stream SendMsg is not concurrent safe)
- add a send buffer for immediate return after calling Send
- flow control. Dynamically delay the caller time when the send buffer is 50% full and slowly reduce the delay when the buffer empties out.
- if the send buffer is full then add an extra 10usec delay and repeat up to 10 times (configurable). After that returns ErrClientTooSlow error.

No need for protobuf magic and types and stuff. Simply add the stream on the server and connect to it on the client.
This registers the 'jsonCodec' in case of complex objects. Or the user can simply send and receive string or []byte arrays and use their own marshalling codec.

Overhead is negligable. On an intel i5 4570S, 2.9GHz this transfers 300K 300byte messages/sec or 250K 1K messages/sec using unix sockets.

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
        "unix:///run/myapp.sock", nil, time.Minute, "myservicename", clientStreamHandler)
	err := grpcClient.ConnectWithToken(clientID, token)
	_, err = grpcClient.Ping("")
	strm, err = grpcClient.ConnectStream("stream-1")
    // this blocks
	strm.WaitUntilDisconnect(name)
```

To use TCP network socket using net.Listen with "tcp", ":port", and client URL "dns:///localhost:port". // Note the triple forward slashes that gRPC uses when no DNS server is provided.
This is intended for educational and experimental purpose. It is recommended to use the module instead.

## Usage

1. Create and start the server module

```go
srv := NewHiveotGrpcServer(connectURL, tlsCert, authn, respTimeout)
srv.Start()
```

2a. Link it to an agent for serving requests, when running a standalone device

```go
// create the agent and link it to the server
agent := NewAgent(clientID)
srv.SetRequestSink(agent.HandleRequest)
agent.SetNotificationSink(srv.HandleNotification)
// set the request handler
agent.SetAppRequestHandler(myapphandler)
// publish updates
agent.PubEvent(thingID, eventName, value)
agent.PubProperty(thingID, propName, value)
```

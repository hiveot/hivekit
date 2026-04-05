# grpc - HiveOT gRPC Transport

'[grpc](https://grpc.io/)' is a high performance Remote Procedure Call framework that can run in any environment.

This transport module passes RRN messages as-is between client and server using a bi-directional gRPC stream.

## Status

This transport module is functional but breaking changes can be expected.

TODO-1: Support for tcp sockets and URL
TODO-2: change stream name
TODO-3: use a separate notification stream
TODO-4: fix race conditions

## Summary

The primary purpose of the gRPC transport is to support Unix Domain Sockets for high performance (10K msg/sec on rPi3+, 100K/sec on i5) local inter-process communication. Tcp sockets are also supported, expanding its use to the network.

It is mainly intended for agents that use reverse connections Since they represent multiple devices or services, performance is more important than with a single isolated device.

## Configuration

The module configuration consists of:

1. The connection URL which indicates the scheme (unix or tcp) and the socket or tcp address.
2. A TLS certificate on the server and CA certificate on the client.
3. An authenticator for validating client credentials. This is optional when using UDS.
4. Optional timeout to facilitate testing and debugging. The default is defined in DefaultRpcTimeout.

## Usage

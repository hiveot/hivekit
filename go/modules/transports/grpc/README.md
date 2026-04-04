# grpc - HiveOT gRPC Transport

'[grpc](https://grpc.io/)' is a high performance Remote Procedure Call framework that can run in any environment.

This transport module passes RRN messages as-is between gRPC client and server.

## Status

This transport module is functional but breaking changes can be expected.

TODO-1: Support for tcp sockets and URL
TODO-2: Remove protobuf, just use gRPC. This currently uses gRPC protobuf to generate code for a single stream. Protobuf just adds an encoding to the existing json encoding so it isn't really helping in any way. It also adds another toolset dependency. It will be replaced with a non-protobuf encoder once the code is complete.

## Summary

The primary purpose of the gRPC transport is to support Unix Domain Sockets for high performance (10K msg/sec on rPi3+, 100K on i5) local inter-process communication. Tcp sockets are also supported, expanding its use to the network.

It is mainly intended for agents that use reverse connections Since they represent multiple devices or services, performance is more important than with a single isolated device.

## Configuration

The module configuration consists of:

1. The connection URL which indicates the scheme (unix or tcp) and the socket or tcp address.
2. A TLS certificate on the server and CA certificate on the client.
3. An authenticator for validating client credentials. This is optional when using UDS.
4. Optional timeout to facilitate testing and debugging. The default is defined in DefaultRpcTimeout.

## Usage

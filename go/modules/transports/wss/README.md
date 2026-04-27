# Websocket Transport Module

The websocket transport module provides a full bi-directional asynchronous messaging between client and server using websockets over http/1.1.

!!! This module will be replaced with websockets over http/2 and will have an identical external facing API.

This module supports two sub-protocols:

1. [WoT Websocket protocol](https://w3c.github.io/web-thing-protocol/). This is an offical WoT http subprotocol following the WoT websocket specification.

2. The HiveOT websocket protocol which works akin to the WoT websocket protocol but passes the RRN message envelopes as-is, instead of converting them to a more complicated message format. This makes it slightly more efficient. This is intended for connections that do not require WoT interoperability.

Connecting over websockets requires a valid bearer token in the http authorization header.

## Status

The Websocket Transport module is in alpha. It is functional but breaking changes might still be possible.

A drop-in replacement using http/2 for bidirectional streams is planned for the future. This module will remain available for the forseeable future.

## HTTP Server Dependency

This module uses a http server that implements the IHttpServer interface. HiveKit includes the 'httpserver' module which implements this interface. This interface only has a few methods including two for getting public and protected (chi) routes, so it is easy to whip up an alternative module for this if needed.

## Summary

This module can both receive and send messages over established websocket connections.

Websocket Transport connections implement the IConnection interface which contains handlers for receiving and sending, request, response and notification messages. A connection can therefore be used as a consumer or device agent.

This module uses the TransportServerBase library that takes care of managing multiple incoming connections.

This module supports the AddTDSecForms method that updates TDs with the security information and forms to connect to the transport server.

## Configuration

The module configuration includes:

1. wsspath - the websocket connection path. WoT transports default to /wot/wss and HiveOT transports use /hiveot/wss.
2. The embedded HTTP server which is configured for certificates and authentication.
3. Optional timeout to facilitate testing and debugging. The default is defined in DefaultRpcTimeout.

## Usage

There are two ways to create a websocket transport module instance, manually or using the pipeline factory.

For manual instantiation call wsstransport.NewWoTWssTransport or NewHiveotWssTransport, and provide it an embedded http server. The http server must implement the transports.IHttpServer interface. The transports.httpserver package can be used with any transport that uses http.

When using the module pipeline factory, the module is automatically instantiated using the pipeline configuration and linked to the configured http server.

To create the websocket transport client, use wsstransportclient.NewWotWssTransportClient, or NewHiveotWssTransportClient. This requires a websocket URL and a CA certificate to validate the connection. The http server certificate must match this CA. The URL can be obtained using discovery, or provided through configuration. This is up to the application developer. The module pipeline also contains a recipe for discovering servers and creating a client for connecting to the discovered server. (\* still todo)

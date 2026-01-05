# WoT Websock Transport Module

The Wot Websocket Transport Module implements the WoT Websocket protocol. It allows access to WoT IoT devices using websocket messages and emits these messages using RRN (Request-Response-Notification) messages.

## Dependencies

This module depends on IHttpServer interface, which can be provided by any compatible http server implementation such as the 'httpserver' module. This interface only has 3 methods of which two are routers, so it is easy to whip up an alternative module for this if needed.

## Summary

This module can both receive and send messages via the server incoming connections. The connections are passed to the connections callback handler which must track these connections.

Each connection contains callback handlers for receiving request, response and notification messages. A connection can also send these messages to the remote endpoint.

This module is best paired with the Connections Module, which takes on the task of managing multiple incoming connections and receive the messages from these connections.

If this module is used stand-alone, the application is responsible for registering the connection callback, managing the connections and listen for messages.

## Configuration

The module configuration includes:

- wsspath - the websocket connection path. Defaults to /wot/wss

## Usage

There are two ways to create a WoT websocket module instance, manually or using the pipeline factory.

Consumers can connect using the WotWssClient, send requests and

### Pipeline Factory

When using the pipeline factory, the module is automatically instantiated using the pipeline configuration and linked to the configured http server and connection module.

### Manual Module Creation

This just needs a http server to run, and optionally a handler to receive connections.

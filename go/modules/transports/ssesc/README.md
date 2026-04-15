# HiveOT SSE-SC Transport Module

The HiveOT SSE-SC transport module provides a full bi-directional asynchronous messaging between client and server using HTTP for sending messages to the server and SSE for receiving messages from the server. This is a http subprotocol following the HiveOT sse-sc specification.

Connecting over SSE-SC requires a valid bearer token in the http authorization header.

## Status

SSE-SC is in alpha. It is functional but breaking changes can be expected.

## Overview

The SSE-SC transport module adds endpoints to a http server for publishing RRN (Request-Response-Notification) messages and to open an SSE return connection for receiving these messages.

SSE-SC refers to a single-connection SSE transport protocol. This transport supports full asynchronous messaging using HTTP/SSE but is not a WoT standard.

This SSE protocol implementation is not a WoT specified protocol. It is however easier and more efficient to use, requiring only a single connection to subscribe and receive messages. It uses the hiveot RequestMessage, ResponseMessage and NotificationMessage envelopes for all messaging.

This transport differs from the WoT SSE specification in that a single SSE connection can be used to subscribe to any events and observe any properties of a Thing or multiple Things, instead of having to open a new SSE connection for each subscription.

This can be used with any golang HTTP server, including the provided httpserver module.

Usages:

1. Thing Agents that run servers.
   For example an IoT device, service or gateway. The agent serves HTTP/SSE connections from consumers. Requests are received over HTTP and asynchronous responses are sent back over SSE. HTTP requests and SSE connections must carry the same 'cid' (connection-ID) header to correlate HTTP requests with the SSE return channel from the same client. The HiveOT Hub uses this as part of multiple servers that serve the digital twin repository content.

2. Consumers that run servers.
   For example the Hub is a consumer of Thing agents that connect to the Hub. Since the connection is reversed, the requests are now sent over SSE to the Hub while the response is sent as a HTTP post to the hub.

3. An agent/consumer hybrid that runs a server. For example, the HiveOT Hub.
   Another Thing agent or service connect to the Hub to receive requests and at the same time can send consumer requests over http and receive responses over SSE.

Note that the direction of connection is independent from the role of consumer or agent.

All SSE messages use the 'event' and 'data' field as per SSE standard. The SSE 'event' field contains the message type, request, response or notification, while the data field contains the JSON serialized RequestMessage or ResponseMessage envelope.

See the test cases for example on how to use this module in the code.

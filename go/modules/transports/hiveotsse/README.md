# HiveOT HTTP-SSE Transport Module

Hiveot HTTP-SSE Server is a transport server for both HTTP, and the SSE-SC sub-protocol. SSE-SC is refers to a single-connection SSE protocol. This transport supports full asynchronous messaging using http/SSE but is not a WoT standard.

Note: The use of SSE is optional. This binding still serves an important role as the HTTPP protocol binding, and provides login and token refresh endpoints.

This SSE protocol implementation is not a WoT specified protocol. It is however easier and more efficient to use, requiring only a single connection. It uses the hiveot RequestMessage and ResponseMessage envelopes for all messaging, or an alternative message converter can be provided to support a different message
envelope format.

This can be used with any golang HTTP server, including the http-basic or websocket http server as long as it can register routes.

Usages:

1. Thing Agents that run servers.
   For example an IoT device, service or gateway. The agent serves HTTP/SSE connections from consumers. Requests are received over HTTP and asynchronous responses are sent back over SSE. HTTP requests and SSE connections must carry the same 'cid' (connection-ID) header to correlate HTTP requests with the SSE return channel from the same client. The HiveOT Hub uses this as part of multiple servers that serve the digital twin repository content.

2. Consumers that run servers.
   For example the Hub is a consumer of Thing agents that connect to the Hub. Since the connection is reversed, the requests are now sent over SSE to the Hub while the response is sent as a HTTP post to the hub.

3. An agent/consumer hybrid that runs a server. For example, the HiveOT Hub.
   Another Thing agent or service connect to the Hub to receive requests and at the same time can send consumer requests over http and receive responses over SSE.

Note that the direction of connection is independent from the role of consumer or agent.

All SSE messages use the 'event' and 'data' field as per SSE standard. The SSE 'event' field contains the message type, request, response or notification, while the data field contains the JSON serialized RequestMessage or ResponseMessage envelope.

# HiveKit Transport Modules

## Summary

HiveKit transport modules provide client and server modules for connecting producers and consumers running on different devices or in different processes. Both the client and server modules can be linked to any other module that needs to send requests, receive notifications or vice-versa. Transport modules are implemented for various IoT protocols, and link with modules using the HiveOT standard RRN message envelope. Common transport modules support WoT protocols http-basic, websocket, the hiveot http/sse-sc protocol. MQTT is under development.

All client and server modules implement the IHiveModule interface to support chaining modules in a pipeline. This supports the following use-cases:

1. A consumer links to a client module to send requests and receive notifications.
2. An IoT device or service is linked-to by a server module to receive requests and return responses and notifications.
3. An IoT device links to a client module - using connection reversal - to receive requests and publish notifications.

Transport module support sending requests from client to server and vice versa. Similarly, responses and notifications can be sent by the client and by the server side. Therefore the producer and consumer role is separated from the client and server role. This enables connection reversal and allows IoT devices to connect to a gateway that handles things like authentication and authorization.

This is achieved by requiring that the HandleRequest and HandleNotification methods of a (client and server) Transport module sends the message to the remote side of the connection. Received messages are passed to the registered request sink and notification sink.

Subscriptions are handled by the server side depending on the protocol used. When a consumer subscribes to notifications, the request is forwarded until it reaches a server module which tracks the notification and links it to the connection.

## Uni-Directional vs Bi-Directional Transports

Most protocols are bi-directional in that responses and notifications can be sent from server to client.

The HTTP-Basic protocol is an exception as http is connectionless and uni-directional. In this context it means that messages can be received server side from remote clients but not pushed to remote clients. With HTTP-basic, a response can be returned to the remote client if it is received by the server module before the request completes.

The HiveOT SSE-SC module extends HTTP-Basic with an SSE return channel. This enables it to be used as a bi-directional protocol just like websockets. In most cases the websocket protocol is preferred as it is a WoT standard.

Subscription to events and property updates are therefore not supported by uni-direction protocols. Remote clients will have to 'poll' for information to receive updates.

Bi-directional protocols, such as websockets and mqtt support both sending and receiving messages on the client and server side. Consumers can subscribe to events and property changes.

## Synchronous vs Asynchronous messaging

Synchronous messaging can be easier to use than asynchronous because the result of a request is available when the request completes. Asynchronous request/response messaging lets responses be sent separate from the request. The request immediately returns once it is delivered at the first destination.

While the benefit of synchronous messaging is simplicity in use, one of the main downsides is that it blocks the resources involved in the request until it completes. This increases resource use and reduces performance. This is aggravated when using a gateway where the request travels from consumer to gateway and on to device.

Asynchronous messaging has the potential for greater throughput. The downside of using asynchronous messages is that the return path must be determined in order to deliver the response. This is further complicated when using an intermediate gateway. When a transformation takes place in the request, the response might also need to be converted which puts restraints on the return path. Furthermore debugging is more difficult when corrupted or missing responses are not easily correlated to requests.

Most HiveOT transports work fully asynchronously to maximize performance and keep resource usage to a minimum.

The HiveOT approach is to provide a 'replyTo' callback in the HandleRequest API that can be called at any time to return a response. The response path is therefore known and can be handled asynchronously depending on the transport protocol.

## Client-Server vs Consumer-Agent (Connection Reversal)

One of the objectives of Transport modules is to support connection reversal. During connection reversal an IoT agent (device, service) connects as a client to a protocol server. Once connected it accepts requests and sends responses and notifications just as if it was running a server. The IoT agent functions independently from how the connection is established.

### Why Connection Reversal

The primarily benefit of using connection reversal is that it is much more resistent against hacking as devices cannot be directly accessed.

Secondary benefits are that authentication and authorization can be centralized in a gateway, the user interface provided by a gateway is consistent for all IoT devices, and monitoring and logging can be centralized. New automation features can be added to the gateway even if the device doesn't support them.

Last but not least, this removes quite a burden from IoT devices as they now only have a single purpose: Interface with hardware or a service. They don't have to manage users, don't run a server, and don't implement a user interface for day-to-day usage. This results in lower memory and cpu requirements.

The main downside of using connection reversal is that it needs a gateway that devices discover and connect to. There is currently no clearly defined standard to accomplish this, although WoT discovery can be used to discover the gateway as if it is an IoT device.

### Supporting Connection Reversal

The good news for supporting connection reversal is that it is quite easy to support. Transport modules just need to transfer all request, response and notification messages regardless if are send by via the client or from the server.

The main 'burden' is on the IoT device side that instead of listening for incoming connections, it now must discover and connect to a gateway.

Note that not all WoT protocols, such as WoT HTTP-Basic/SSE, support connection reversal as they need to describe both the consumer side and device side of the interaction. The websocket protocol on the other hand defines both consumer and device message payloads and is a suitable option for connection reversal. Any protocol that can be converted into RRN request, response and notification messages are suitable to be used with connection reversal.

## Transport API

The server side module of a transport must implement the ITransportServer interface. The client side must implement the IHiveModule and IConnection interfaces.
By doing so they can be used by any module sending requests and receiving notifications.

## Subscriptions

In HiveKit subscriptions are the responsibility of the transport server module. There are several reasons for this:

1. The client-server communication is often the most costly wrt performance.
2. Some protocols such as MQTT already have built-in support for subscription.
3. The classic use-case where the IoT device runs a server works out of the box when using the HiveKit server module. There is no need for it to manage subscriptions as the server module takes care of it.
4. It is the most efficient for the gateway use-case. When using a gateway IoT devices can serve many consumers via the gateway. Handling subscriptions at the gateway server avoids having to send notifications multiple times, one for each subscriber, from the IoT device to the gateway.
5. Middleware modules such as logging, history storage and other types of transformations often need access to the full data stream. Therefore the IoT device ends up sending all (notification) messages anyways.

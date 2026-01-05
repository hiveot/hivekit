# HiveKit Transport Modules Considerations

HiveKit transport modules are intended for connecting IoT consumers with IoT agents (sources) such as devices and services. The transport module converts between the protocol message format and the HiveKit's standard RRN message envelope.

All modules implement the IHiveModule interface to support interaction between modules. Sinks can be added to use the module as a source. HandleRequest/Response/Notification methods allow the module to be used as a sink.

A transport module contains a service which implements the protocol server, an API that converts protocol messages to RRN envelopes and vice versa, and a module that implements the IHiveModule interface.

Commonly used protocols with WoT compatibility specifications are HTTP-Basic, Websocket, MQTT, CoAP. Support for non-wot communication protocols such as UDS (sockets), gRPC, and such can easily be added.

## Uni-Directional vs Bi-Directional Transports

The HTTP-Basic protocol is connectionless and uni-directional. In this context it means that messages can be received from remote clients but not pushed to remote clients. With HTTP, a response can be returned to the remote client.

Subscription to events and property updates are therefore not supported by uni-direction protocols. Remote clients will have to 'poll' for information to receive updates.

Bi-directional protocols, such as websockets and mqtt support both sending and receiving messages on the client and server side. Consumers can subscribe to events and property changes.

Note the difference between client-server and consumer-agent.

## Synchronous vs Asynchronous messaging

Synchronous messaging is easier to use than asynchronous because the result of a request can simply be returned to the caller. Asynchronous request/response messaging lets responses be sent separate from the request. The request immediately returns once it is delivered at the first destination.

The benefit of synchronous messaging is simplicity in use. One downside is that it can block the resources involved in the request until it completes. This increases resource use and reduces performance. This is aggravated when using a gateway where the request travels from consumer to gateway and on to device.

Asynchronous messaging has the potential for greater throughput. The downside of using asynchronous messages is that the return path must be determined in order to deliver the response. This is further complicated when using an intermediate gateway. When a transformation takes place in the request, the response might also need to be converted appropriately putting restraints on the return path. Furthermore debugging is more difficult when corrupted or missing responses are not easily correlated to requests.

Most HiveOT transports work fully asynchronously to maximize performance and keep resource usage to a minimum.

Option 1 - chain
Modules include a 'replyTo' parameter in the handleRequest API. When a response is available it is passed to the replyTo handler. The implementation of the replyTo handler determines whether the response is send synchronously or asynchronously, decoupling it from the module. This is similar to chain where the response travels in the reverse direction.

The main difference with the other options is that the replyTo can be determined dynamically based on the message.

Option 2 - pipe
Modules always respond by invoking the response handler on sinks. The sinks determines where it goes. From the module point of view the messaging flow is uni-directional.

Module sinks do not change per message. As such the direction of the response is statically determined unless a router module is used. Router modules support multiple sinks and can dynamically determine which one a RRN message is directed to.

Option 3 - star
Modules are all linked to a router that receive all messages from all in-process modules and passes it to the appropriate module based on configuration.
This star configuration is open to both static and dynamic usage. The downside of this approach is that it requires a 'star' module, even in simple setups. It forces a centralized message flow. - thoughts ... maybe this approach fits best in this problem domain.

How does a module subscribe to notifications from another module/service? Transport modules handle this server-side for remote clients, but now we introduced in-process modules that interconnect.
option 1: modules can use a connection instance that represents the module as a consumer. This direct-connection handles subscription just like transports.
option 2: modules don't subscribe, until a use-case presents itself. Then see option 1.
option 3: [star] the router module handles all internal subscriptions.

// when responses don't need the same return path:
consumer -> [client] -> [server] -> agent
consumer -> [client] -> [server] -> [authoriation] -> agent
consumer -> [client] -> [server] -> [authoriation] -> [logging] -> agent
consumer -> [client] -> [server] -> [authoriation] -> [logging] -> [rate_control] -> agent
consumer -> [...] -> [filter] -> agent
consumer -> [...] -> [conditional_switch] -> agent
consumer -> [...] -> [bridge] -> [consumer] -> [server] -> agent

// service modules
[automation] => (can generate additional requests and optionally cancel the original)
[presentation]
[digital_twin] duplicates a Thing state and present it as a new Thing (performance improvements)

// when responses might follow the same return path?
consumer -> [client] -> [server] -> [...] -> [latency] <-> agent
consumer -> [client] -> [server] -> [...] -> [history_store] <-> agent

## Client-Server vs Consumer-Agent (Connection Reversal)

One of the objectives of Transport modules is to support connection reversal. During connection reversal an IoT agent (device, service) connects as a client to a protocol server. Once connected it accepts requests and sends responses and notifications.

### Why Connectionn Reversal

The primarily benefit of using connection reversal is that it is much more resistent against hacking as devices cannot be directly accessed.

Secondary benefits are that authentication and authorization can be centralized in the gateway, the user interface is consistent across all devices, monitoring and logging can also be centralized. New automation features can be added even if the device doesn't support them.

Last but not least, this removes quite a burden from IoT devices as they now only have a single purpose: Interface with hardware or a service. They don't have to manage users, don't run a server, and don't implement a user interface for day-to-day usage. This results in lower memory and cpu requirements.

The main downside of using connection reversal is that it needs a gateway that devices discover and connect to. There is currently no clearly defined standard to accomplish this, although WoT discvoery can be used to discover the gateway as if it is an IoT device.

### Supporting Connection Reversal

There is little to do for Transport protocols to support connection reversal, just accept all messages regardless if they are from consumers or IoT devices. Eg, send and receive requests, send and receive responses and send and receive notifications.

The main 'burden' is on the IoT device side that instead of listening for incoming connections now discovers and connect to a gateway. While there are no existing devices that support this, it is an excellent oppertunity to add this to 3rd party protocol bindings such as ZWave, Insteon, Zigbee, Philips Hue, LoRaWan, and so on.

Note that not all WoT protocols, such as WoT HTTP-Basic/SSE, support connection reversal as they need to describe both the consumer side and device side of the interaction. The websocket protocol on the other hand defines both consumer and device message payloads and is a suitable option for connection reversal. Any protocol that can be converted into RRN request, response and notification messages are suitable to be used with connection reversal.

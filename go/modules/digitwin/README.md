# digitwin - Digital Twin Module

The objective of the digital twin concept is to have consumers communicate with digital twins instead of the actual devices. Communication between digital twin and device remains hidden from consumers. The digital twin publishes an updated TD for use by consumers that describes how to interact with the digital twin to read properties, subscribe to events, update configuration and invoke actions.

This approach provides the following benefits:

1. Improved security. Devices remain isolated from consumers. Many types of security vulnerabilities can not be utilized if the device cannot be reached directly.
2. Thing state remains available even when a device is not reachable, like when it entered sleep mode or when its connection is intermittent.
3. Access to devices no longer requires the consumer to use a variety of transport protocols and endpoints. Instead, a single protocol can be used to connect to the digital twin server.
4. Simplified provisioning. Consumers only need a single account to access all devices. The devices only need to be provisioned with a digital twin account. This avoids the need to create consumer accounts on each device.

## Status

This module is in development. It is partial functional but incomplete. Breaking changes can be expected.

TODO:

1. send notifications to consumers when digital twin device state changes or events are received
1. detect agent (dis)connection with server to show online status of devices
1. track online status of devices
1. connect and subscribe to known WoT devices on startup and when they are discovered
1. test integration with router module for forwarding requests
1. test OOB (out of band) provisioning by admin through upload of device TDs

## Summary

Using HiveKit in an application involves an interplay of a few hivekit modules in order to serve a digital twin directory, read cached values, and write properties and invoke actions. This is described in more detail below:

These modules are linked in a pipeline:

```
 [1:http server]  ─────────┐
  ├─ [2:discovery]         ↓  ┌─────(5)─────┐  ┌─────(5)───┐
  ├─ [3:msg servers] -> [4:directory] -> [5:digitwin] -> [6:router]
                                            │       |         |
                                       [vcache]  [device-directory]

     [router] -> [clients]     (7. wot clients)
              -> [msg server]  (8. reverse connections)

```

(1) The http server is used by discovery, the directory, and messaging server. It authenticates and identifies connecting clients and routes http requests to the endpoint registered by the modules.
(2) The discovery server publishes the TDD as provided by a Thing directory using DNS-SD.
(3) The messaging transport server(s) serves non-http messaging protocol such as websocket and mqtt. This transport receives property read, write and action requests from consumers, and property and event change notifications sent by agents.
(4) The directory handles request messages for reading and writing the directory as included in the TDD forms. Optionally it also offers an HTTP API for the same as per specification.
(5) The digitwin module hooks into the directory to intercept directory write requests and replace TD's with a digital twin TD whose forms point to the digitwin module. All read/write requests for digital twins are handled by the digitwin module using the value cache module.

- The original device TD's are stored in a separate device directory for use by the router.
- Notifications received from devices with a digital twin are stored in a vcache.

(6) The router passes messaging requests to devices using the device TD.
(7) The router establishes client connections for passing messages to standard WoT compatible devices.
(8) The router uses the messaging server for passing messages to devices whose agents have a reverse connection.

Hiveot uses the concept of 'agents'. Agents are services that manage one or multiple Things. For example, a 1-wire bus can have up to 63 devices connected. The service that manages the 1-wire bus therefore represents up to 63 Things. It can create up to 63 TD's, each containing information on how to connect to the service. Thus, the service is the 'agent' for the 63 devices. When the term 'agent' is used it therefore refers to the service and not the Things that are managed by the agent. HiveOT agents use reverse connection to adhere to the 'Things dont run servers paradigm'.

### [discovery](../transports/discovery/README.md)

The [discovery module](../transports/discovery/README.md) publishes the availability of the TDD (Thing Description Directory) on the local network using DNS-SD. This follows the WoT discovery specification.

Devices, Agents and Consumers kickstart their application or service by looking for the directory using discovery. When found, the discovery record contains the URL to the TDD. This TDD describes how to write a TD in the directory for use by devices or agents, and how to read the directory for use by consumers.

### [directory](../directory/README.md)

The [directory module](../directory/README.md) supports both HTTP methods and WoT messaging protocols for writing and reading the directory as per specification. The endpoints are described in the TDD that is shared through discovery.

The use of the RESTful HTTP API is optional. The use of messaging protocols such as websockets and mqtt are preferred for larger systems that include pipeline modules for request logging, authentication, authorization and others. The TDD will include forms for the enabled protocols.

The HiveKit directory module supports a hook that is invoked when a device writes its TD, just before it is actually stored. This hook is used by the digitwin module to intercept the request.

### digitwin (this module)

The digital twin module hooks into the directory to receive a callback each time a TD is written. If the Thing is determined to have a digital twin, the TD is used to create a digital twin TD which is then returned to the directory. The directory then stores the digital twin TD instead of the device TD. The device TD is stored in a device directory which is part of the Digitwin module.
All digital twin Things include observable properties indicating whether the device is online and when it was last seen.

Consumers receive the digital twin and non-digital-twin TD's that are stored in the [directory](../directory/README.md). The digital twin TD is a modified copy of the device TD. TD Forms that describe the protocol used to interact with the device are replaced with forms that now point to the digital twin instead. The ThingID is modified with a "dtw:" prefix, since the digital twin is a different Thing so it must have a different ID. The default protocol used is the WoT websocket protocol. Additional protocols can be enabled by including the protocol transport module.

Not all devices have a digital twin. Application services, including the discovery service, automation services, storage services, and others, can publish a TD that describes how to use the service. These are not IoT devices and should not have a digital twin. Service TD's are identified by looking at the @type field of the TD. HiveOT defines the "service" value for services. Since WoT does not define a vocabulary for the @type field, this needs to be configurable with other values.

The digital twin module will receive requests for reading properties and events, writing properties, and invoking actions. Reading properties is handled by the vcache module described below.

When requests to write properties and invoke actions are received they must be passed to the actual device. A copy of the request is modified to contain the actual device Thing ID and forwarded to the router. The response will be passed on to the caller, after the Thing ID of the response is converted to the digital twin ID. In case of Invoke action, the action status is stored in the vcache module so it can serve action status queries.

A future improvement can be to support a request validity period during which the request can be held untile the device is reachable. The response to such requests should have the status set to pending delivery.

See also the router module described below which handles delivery of requests to the actual device.

### [vcache](../vcache/README.md) - notification value cache

#### Updating values in the vcache

The digital twin has two methods in which it receives property and event notifications for caching in the digital twin.

a. Subscription to registered WoT devices.

When a TD is written to the directory and it receives a digital twin, the module makes a request to subscribe to all its events and observe all property changes. This request is forwarded to the next module in the chain which is typically a router. The router establishes a connection if needed and passes on the subscription. The same connection will receive the notifications, which the router returns to the digitwin module. (notifications travel backwards in the pipeline). The digitwin module passes them to the vcache which will store the last notification of each affordance.

On restart, the module re-subscribes to all notifications of devices that have a digital twin.

b. Notification push by agents that use reverse connection.

Thing agents establish a connection to the server described in the directory TDD they discovered. Agents publish notifications to the server without using subscriptions. When the server receives a notification it passes this to the registered notification handler. Since notifications travel in reverse direction of requests, this is typically the end of the chain, eg the router.

The router forwards the notification to the digitwin module, similar to notifications received from one of its connections.

#### Reading values from the vcache

Requests to read digital twin device properties, events, or queries for action status are first sent to the digital twin which passes it on to the vcache module. The vcache module responds with the cached value, if available.

If the vcache module does not hold the requested values, it cannot respond immediately with a result. Instead the request is forwarded to its sink, which is set to a digitwin module handler. Therefor the request loops back if vcache cannot fulfil it. After modifying the thingID in the request to its original value the handler will forward the request to the original device using the [router module](../router/README.md). Once a response is received, it is passed back to the vcache, which then returns it to the caller.

Only observable properties can be served by the vcache module as non-observable properties are not send as notifications and can be out of date. Thefore it is more efficient to query observable and non-observable properties separately.

Note that the TD of devices or services that do not have a digital twin will remain unchanged so it will contain forms that point to the actual device or service. Requests to these Things do not pass through the digital twin server.

### [Router](../router/README.md)

The request sink of the digital twin module is set to the router module. Requests not handled by the digital twin module are forwarded to the router which handles further delivery. This can happen in one of two situations. First, the request is forwarded by the digital twin when only the actual device can handle it, like a write property request. Second, the request is for an external device or service that does not have a digital twin. In this case the request originates from one of the internal modules that are part of the server side pipeline.

The digital twin module forwards requests to subscribe to events, observe properties, write properties, invoke actions and read unobservable properties to the router.

The [router module](../router/README.md) must determine how to deliver these requests. It does this with help of the device directory that is managed by the digital twin module. The router looks up the TD and the form for handling the request. The form contains the protocol and protocol for sending the request to the device.

In case of standard Thing devices, the router will establish a connection to the device, or re-use an existing connection, and pass the request.

In case the device is managed by an agent that uses reverse connection, the router forwards the request to the server that has that connection. Reverse connections are not described in the WoT specifications so they only works for HiveOT compatible Thing agents.

How does the router know a Thing is accessed via an agent with reverse connection?
Agents discover the directory TDD just like devices. They connect to the server defined in the base attribute of the directory TDD, using the compatible protocol identified by the schema. By default this is the websocket connection endpoint.

When agents write TDs to the directory, either through http or using the messaging protocol, the directory receives the agent ID along with the TD. The agent ID is set in the root form of the device TD and stored as part of the TD in the device directory (not the digital twin directory). When the router looks up the form in the TD of the device to forward a request to, its form describes the protocol as a reverse connection from the agent. While this is not a WoT specification, the domain knowledge for this mechanism is limited to the digitwin and router modules and has no external dependencies.

## Usage

This module is designed to be used with a protocol server, a thing directory, and a router module. In addition, the use of a discovery, authentication, authorization modules are highly recommended. Furthermore the logging module can be useful to trace the messages passing through the application.

A basic setup could use modules as follows:

### Request Pipeline

Below a description how a request pipeline handles the various requests to the directory and the digital twin. [rsink] follows a registered request sink and [call] follows a registered callback hook.

1. Request flow for writing the directory, made by agents with reverse connections:

> server -[rsink]> directory -[call]> digitwin

2. Request flow for reading the directory, made by consumers connect to the server

> server -[rsink]> directory

3. Request flow to read digitwin values, made by consumers. (since modules can only have one sink, these requests flow through prior modules in the chain)

> server -[rsink]> directory -[rsink]> digitwin -[call]> vcache

4. Request flow to read uncached digitwin values, made by consumers, where the digital twin is reachable via an agent. The digitwin module maps from digital twin to device IDs.

> server -[rsink]> directory -[rsink]> digitwin -[rsink]> router -[rsink]> server => agent

5. Request flow to read uncached digitwin values, made by consumers, where the digital twin is reachable via a client connection made by the router. The digitwin module maps from digital twin to device IDs.

> server -[rsink]> directory -[rsink]> digitwin -[rsink]> router -[rsink]> client => device

As is shown, these various usages can all be handled using the same pipeline. Each module decides whether to answer a request or forward it to the next module.

### Notification Pipeline

Notifications originate devices and services and are consumed by consumers and services such as the vcache module.

[nsink] follows a registered notification sink

1. Notification flow from agent to digital twin. This notification is received by the server and forwarded by the router to updates the vcache of the digital twin. The digitwin module maps the device ID to the digital twin ID before updating the vcache.

> server -[nsink]> router -[nsink]> digitwin -[nsink]> vcache

2. Notification flow from subscribed device to digital twin. This updates the vcache of the digital twin. The client is a connection established by the router.

> client -[nsink]> router -[nsink]> digitwin -[nsink]> vcache

3. Notification flow from the digital twin vcache to consumer. When a digital twin value is updated in the vcache a notification is sent to consumers. Consumer subscription is managed by the server.

> vcache -[nsink]> server

4. Notification flow from directory to consumer. When a TD is created/updated/deleted in the (digital twin) Thing directory, a notification is sent to consumers.

> directory -[nsink]> server

5. A single pipeline that supports use-cases 1-4. When a notification is forwarded to the server it is sent to connected clients that have subscribed to it:

> server -[nsink]-> router -[nsink]-> digitwin -[nsink]-> vcache -[nsink]-> directory -[nsink]-> server

See also [test/Digitwin_test.go](test/Digitwin_test.go) for an example of setting up this pipeline.

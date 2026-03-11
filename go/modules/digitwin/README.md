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

1. detect agent (dis)connection with server
1. subscribe to agent notifications after they connect
1. track online status of devices
1. connect and subscribe to known WoT devices on startup and when they are discovered
1. send notifications to consumers when digital twin device state changes or events are received
1. test integration with router module for forwarding requests
1. test OOB (out of band) provisioning by admin through upload of device TDs

## Summary

Using HiveKit in an application involves an interplay of a few hivekit modules in order to serve a digital twin directory, read cached values, and write properties and invoke actions. This is described in more detail below:

These modules are linked in a pipeline:

```
 [http server]  ───┬─────────────────┐
         │         │  ┌─────(1)────┐ │ ┌─────(2)────┐  ┌─────(3)────┐
 [msg server] -> [discovery] -> [directory] -> [digital-twin] -> [router]
                                                 │        |          |
                                             [vcache]  [device-directory]

       [router] -> [clients]     (4. wot clients)
                -> [msg server]  (5. reverse connections)

```

(1) The discovery service publishes the directory TD using DNS-SD
(2) The digital twin hooks into the directory to intercept directory write requests and replace TD's with a digital twin TD.
(3) The digital twin keeps a copy of the devie TD into a separate device directory for use by the router.
(4) WoT clients connect to WoT compatible devices.
(5) Reverse connections are used by HiveOT agents that manage one or more Things.

The http server is used by the messaging server sub-protocol, the discovery service to provide the directory TDD and the directory to implement the directory API specification.

Hiveot uses the concept of 'agents'. Agents are services that manage one or multiple Things. For example, a 1-wire bus can have up to 63 devices connected. The service that manages the 1-wire bus therefore represents up to 63 Things. It can create up to 63 TD's, each containing information on how to connect to the service. Thus, the service is the 'agent' for the 63 devices. When the term 'agent' is used it therefore refers to the service and not the Things that are managed by the agent. HiveOT agents use reverse connection to adhere to the 'Things dont run servers paradigm'.

### [discovery](../transports/discovery/README.md)

The [discovery module](../transports/discovery/README.md) publishes the availability of the TDD (Thing Description Directory) on the local network using DNS-SD. This follows the WoT discovery specification.

Devices, Agents and Consumers kickstart their application or service by looking for the directory using discovery. When found, the discovery record contains the URL to the TDD. This TDD describes how to write a TD in the directory for use by devices or agents, and how to read the directory for use by consumers.

### [directory](../directory/README.md)

The [directory module](../directory/README.md) supports HTTP methods for writing and reading the directory as per specification.

Devices and agents use the HTTP endpoint or one of the other protocols described in the TDD to write a TD into the directory. This TD includes forms describing how to connect to the device and access the Thing.

The HiveKit directory supports a hook that is invoked when a device writes its TD, just before it is actually stored. This hook is used by the digitwin module to intercept the request.

### digitwin (this module)

The digital twin module hooks into the directory to receive a callback each time a TD is written. If the Thing is determined to have a digital twin, the TD is used to create a digital twin TD which is then returned to the directory. The directory then stores the digital twin TD instead of the device TD. The device TD is stored in a device directory which is part of the Digitwin module.
All digital twin Things include observable properties indicating whether the device is online and when it was last seen.

Consumers receive the digital twin and non-digital-twin TD's that are stored in the [directory](../directory/README.md). The digital twin TD is a modified copy of the device TD. TD Forms that describe the protocol used to interact with the device are replaced with forms that now point to the digital twin instead. The ThingID is modified with a "dtw:" prefix, since the digital twin is a different Thing so it must have a different ID. The default protocol used is the WoT websocket protocol. Additional protocols can be enabled by including the protocol transport module.

Not all devices have a digital twin. Application services, including the discovery service, automation services, storage services, and others, can publish a TD that describes how to use the service. These are not IoT devices and should not have a digital twin. Service TD's are identified by looking at the @type field of the TD. HiveOT defines the "service" value for services. Since WoT does not define a vocabulary for the @type field, this needs to be configurable with other values.

The digital twin module will receive requests for reading properties and events, writing properties, and invoking actions. Reading properties is handled by the vcache module described below.

When requests to write properties and invoke actions are received they must be passed to the actual device. A copy of the request is modified to contain the actual device Thing ID and forwarded to the router. The response will be passed on to the caller, after the Thing ID of the response is converted to the digital twin ID. In case of Invoke action, the action status is stored in the vcache module so it can serve action status queries.

A future improvement can be to support a request validity period during which the request can be held untile the device is reachable. The response to such requests should have the status set to pending delivery.

See also the router module described below which handles delivery of requests to the actual device.

### [vcache](../vcache/README.md) - value cache

When a device or agent writes a TD to the directory and the digitwin service intercepts it, it stores the original device TD in the device directory and subscribes to all observable properties and all events.

As notifications are received the digitwin module uses the [vcache module](../vcache/README.md) to cache the values for later retrieval.

Requests to read a digital twin device properties, events, or queries for action status are first sent to the digital twin which then passes it on to the vcache module. The vcache module responds with the cached value, if available.

If the vcache module does not hold the requested values, it cannot respond immediately with a result. Instead the request is forwarded to its sink, which is set to a digitwin module handler. So the request loops back if vcache cannot fulfil it. After modifying the thingID in the request to its original value the handler will forward the request to the original device using the [router module](../router/README.md). Once a response is received, it is passed back to the vcache, which then returns it to the caller.

Only observable properties can be served by the vcache module as non-observable properties are not send as notifications and can be out of date. Thefore it is more efficient to query observable and non-observable properties separately.

Note that the TD of devices or services that do not have a digital twin will remain unchanged so it will contain forms that point to the actual device or service. Requests to these Things do not pass through the digital twin server.

### [Router](../router/README.md)

The request sink of the digital twin module is set to the router module. Requests not handled by the digital twin module are forwarded to the router which handles further delivery. This can happen in one of two situations. First, the request is forwarded by the digital twin when only the actual device can handle it. Second, the request is for an external service that is used by one of the modules in the pipeline.

The digital twin will forward requests to subscribe to events, observe properties, write properties, invoke actions and read unobservable properties.

The [router module](../router/README.md) must determine how to deliver these requests. It does this with help of the device directory that is managed by the digital twin module. The router looks up the device TD and the form for the request. The form contains the protocol and href for sending the request to the device.

In case of standard Thing devices, the router will establish a connection to the device, or re-use an existing connection, and pass the request.

In case the device is managed by an agent that uses reverse connection, the router forwards the request to the server that has that connection. Reverse connections are not described in the WoT specifications so they only works for HiveOT compatible Thing agents.

How does the router know a Thing is accessed via an agent with reverse connection?
Agents discover the directory TDD just like devices. When writing the directory, either through http or using the websocket messaging, the directory receives the client account ID along with the TD. If the role of the account is "agent" then the Thing is reachable via a reverse connection from that agent. The agent ID is set in the root form of the device TD and stored as part of the TD in the device directory (not the digital twin directory). When the router looks up the form in the TD of the device to forward a request to, its form describes the protocol as a reverse connection from the agent. While this is not a WoT specification, the domain knowledge for this mechanism is limited to the digitwin and router modules and has no external dependencies.

## Usage

This module is designed to be used in a gateway or hub application with connection reversal support for agents.

In addition to the modules in the recipe shown above, some other modules can be useful:

1. An authentication module for managing accounts, authenticate connections and identify roles.
1. An authorization module for authorizing requests based on roles, operations and things.
1. The logging module for logging requests.

Describing in detail how to tie these modules together into an application is out of scope for this module documentation. A separate recipes document is planned with designs for various use-cases.

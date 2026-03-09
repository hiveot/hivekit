# digitwin - Digital Twin Module

The objective of the digital twin concept is to have consumers communicate with digital twins instead of the actual devices. Communication between device and digital twin remains hidden from consumers. The digital twin publishes an updated TD for use by consumers that describes how to interact with the digital twin to read properties, subscribe to events, update configuration and invoke actions.

This approach has the following benefits:

1. Improved security. Devices remain isolated from consumers. Many types of security vulnerabilities can not be utilized if the device cannot be reached directly.
2. Thing state remains available even when a device is not reachable, like when it entered sleep mode or when its connection is intermittent.
3. Access to devices no longer requires the consumer to use a variety of transport protocols and endpoints. Instead, a single protocol can be used to connect to the digital twin server.
4. Simplified provisioning. Consumers only need a single account to access all devices. The devices only need to be provisioned with a digital twin account. This saves time in managing consumers for each device.

## Status

This module is in development. It is migrated from the HiveOT Hub.

## Summary

Hiveot uses the concept of 'agents'. Agents are services that manage one or multiple Things. The agent handles the connection with consumers and publishes device TDs. For example, a 1-wire bus can have up to 63 devices connected. The service that manages the 1-wire bus therefore represents up to 63 Things. It can create up to 63 TD's, each containing information on how to connect to the service. Thus, the service is the 'agent' for the 63 devices. When the term 'agent' is used below it therefore refers to the service and not the Things that are managed by the agent.

Agents play an important role in Thing discovery and in message routing as messages to Things are routed via their agent.

### Device Discovery

The digitwin module needs to determine which devices have a digital twin. This is facilitated using WoT Discovery and the Thing Directory.

Not all devices have a digital twin. Application services, including the discovery service, automation services, storage services, and others, can publish a TD that describes how to use the service. These however are not IoT devices and should not have a digital twin. The digitwin module only creates a digital twin for known IoT devices. The rules for this use the TD attributes and are configurable.

How this works: The digitwin module hooks into the directory module to intercept TD update and delete requests. When agents write the TD's of their devices to this directory, as described in the WoT discovery specification, the digitwin module replaces them with a digital twin version of the TD.

The digital twin TD is a modified copy of the device TD. TD Forms that describe the protocol used to interact with the device are replaced with forms that now point to the digital twin instead. The ThingID is modified with a "dtw:" prefix, since the digital twin is a different Thing so it must have a different ID.

The digitwin module stores the original device TD so it knows how to connect to the actual device for subscribing to notifications and forwarding requests.

When consumers discover devices by reading the directory, they obtain the digital twin TD's.

### Agent Connections

When a digital twin is created, a subscription for events and property updates must be made to the original thing device so the digital twin can remain up to date. In order to subscribe a connection must be established with the device or its agent. This can take place in one of two ways.

The digital twin can connect to a discovered device, or accept a reverse connection from device agents. While the latter is preferred, it is not described in the WoT specifications so only works for HiveOT compatible device agents.

When a TD is received, the digitwin module examines the TD forms to determine if a reverse connection will be used by the agent.

If the TD does not indicate support for a reverse connection, the digitwin module establishes a connection with the device to observe properties and subscribes to events. When subscription is successful the Thing is online. This takes place for each individual Thing.

If the TD does indicate support for a reverse connection, the digitwin module waits for an incoming connection notification, sent by the server module. If the connection is from an device agent, the digitwin module subscribes to the property and event notifications of the Things managed by the agent. Only a single subscription needs to be done as HiveOT agents support wildcard ThingID in subscriptions. This is optional.

All digital twin Things include observable properties indicating whether the device is online and when it was last seen.

### Handling Notifications

When the digitwin module receives notifications from devices, it updates the state of the digital twin properties and events. This in turn causes the digital twin to emit a corresponding notification. These notifications are passed to the server module that is linked as the recipient of the digital twin module notifications.

The server(s) manages subscriptions and forward the notification to those consumers that have subscribed. Consumer subscriptions are not persistent. After the connection breaks the subscription ends. On reconnect a new subscription request must be made by consumers as subscriptions are not persistent.

### Handle Read and Query Requests

When consumers send a read or query request to the digital twin, the server passes passes the request to its linked module pipeline. The pipeline can include additional modules such as authorization and logging. The digitwin module is part of the pipeline and handles all requests directed at digital twin Things.

The digital twin Thing holds a cache with the property, latest event, and action state received from the actual things. Read and query requests are answered using these values.

When the request includes reading a non-observable property, it must be forwarded to the actual thing device to update the cache with the latest value. This will add extra latency as the result cannot be provided until the device responds.

### Handle Property Write Requests

Property write requests must be forwarded to the actual device and returns the success or failure of the request.

If the device is not reachable then this fails.

A future version will improve on this by caching the request and forwarding it when the device is reachable again. Request expiry is needed to ensure they don't linger beyond their intended life span. Request status inquiry is also needed to indicate a request is in progress. Currently the WoT standard does not support request expiry so this will likely be a HiveOT specific solution.

### Handle Invoke Action Requests

Invoke action requests must be forwarded to the actual device and returns the response provided by the device.

If the device is not reachable then this fails.

A future version will improve on this using the same method used for delayed property write requests.

## Agent Connectivity

The digitwin module needs a path to the device agent to send it requests. This path can be established in one or two ways:

1. Connect to the device using the base endpoint described in the Thing TD.

The device runs a server for the protocol(s) described in the TD forms. The digitwin module connects to the device to submit a request. If a connection with the agent already exists then it is re-used. This is the approach described by the WoT standard. The finer details depend on the protocol used.

The digitwin module must have an account and credentials for each device agent.

3rd party WoT enabled devices likely use the first model. Currently not many of these devices exist yet although this number will likely grow in the future.

2. Receive a connection request from the agent when using connection reversal. Once the connection is established the rest of the flow is identical.

In this reverse-connection configuration the agent does not run a server and instead connects to the digital twin server using the account and credentials created on the hub that runs the digitwin module. This model is not standardized by the W3C WoT group.

Hiveot protocol bindings for 3rd party protocols like zwave, insteon, zigbee, etc, use the second model. There is no benefit in using the first model for these protocol bindings as it just reduces the security and takes up more resources. The only downside is that they cannot be used stand-alone with WoT compatible consumers. A future version could make this configurable, eg run a server or use connection reversal.

## Usage

To use this module in a gateway or hub application with connection reversal support for agents, the following additional modules are needed:

1. at least one server for serving consumers and for serving reverse connection of agents
2. a directory module storing and serving digital twins
3. a discovery server for making the directory discoverable (optional)
4. an authentication module for authenticating connections and identify roles (optional)
5. an authorization module for authorizing requests based on roles and configuration (optional)

When supporting wot devices that run servers, eg, not using connection reversal:

6. a transport client for connecting to agents (optional)
7. a discovery client for discovery WoT devices on the network

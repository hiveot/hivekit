# HiveKit Modules

This document provides an overview of available or planned HiveKit pipeline modules. It is divided into the following categories:

- messaging protocol modules for communication
- messaging processing modules such as auth, routing, filtering, rate control, logging
- service modules such as directory, history, etc
- sensor modules for reading sensor data
- actuator modules for controlling actuators

The priority listed here:

- high, all features that the current hub is using
- medium, features already planned or often requested
- low, features that are not requested
- not planned, features that are rely on unavailable hardware or software

## Transport Modules

The role of a transport module is to convert transport protocol messages into RRN messages and vice versa. For example, a WoT WSS (websocket) module can receive WoT compatible websocket messages over TLS and converts them to RRN messages, and returns a response in the WoT WSS format. Most modules contain a server along with a client API.

### HiveOT UDS IPC Transport Module

Status: planned, priority medium

This messaging system uses an UDS (Unix Domain Sockets) based IPC (inter-process communication) protocol to pass messages between client and server running on the same host. This transports messages using RRN messaging.

IPC protocols can be based on shared memory, named pipes, or unix sockets. The initial version will use named pipes.

### HiveOT HTTP/SSE Transport Module

Status: planned, priority low

The HiveOT SSE sub-protocol uses RRN format to pass messages through the SSE connection. This supports subscription to multiple devices, events and properties over a single connection.
This offers more capabilities for web browser client applications than the WoT SSE protocol.

### LoRaWan Transport Module

Status: planned, priority low

The LoRaWan module supports bi-directional messages using the LoRa network.

### WoT CoAP Transport Module

Status: planned, priority low

### WoT HTTP Transport Module

Status: in progress, priority high

The WoT HTTP specification is a limited WoT communication protocol that does not support subscription to properties and events. Client and server modules are provided. The main reason to include this is to allow simple http clients to obtain IoT data.

### WoT MQTT Transport Module

Status: planned, priority medium

The WoT MQTT client and server modules accept bi-directional messages send using the MQTT messaging protocol. Multiple variations exist that can be supported.

### WoT SSE Transport Module

Status: not planned

The HTTP/SSE client and server modules send messages using HTTP and return messages using SSE for the return channel. A SSE connection is required for each subscription. Since better alternatives are available, such as websocket or mqtt, this SSE return channel is not supported.

### WoT Websocket Transport Module

Status: planned, priority high

WoT has recently (2025) released a Websocket standard for sending WoT messages. This can be used to exchange messages between modules, or serve a WoT compatible interface to WoT consumers.

## Message Processing

### Authentication Module

Status: Planned, priority high

The primary purpose of an authentication module is to authenticate a server connection using tokens, certificates or other means. The server module API includes a hook to authenticate incoming connections. The authentication module can plug into this hook.

### Authorization Module

Status: Planned, priority high

The primary purpose of an authorization module is to authorize a message to be forwarded, based on the authenticated sender, operation, agentID, thingID and/or property or action name.

### Dispatcher Module

Status: Planned, priority medium

The primary purpose of a dispatcher module is to provide the handler for a message. The handler can also be another module. The mapping between message and handler can be statically or dynamically configured.

### Filter Module

Status: Planned, priority low

The primary purpose of a filter module is to only forward messages that match the filter parameters, based on message type, device type, operation, agentID, thingID and/or property/event/action name. It can for example be used to determine what messages are passed to a logger.

### Routing Module

Status: Planned, priority high

The primary purpose of a routing module is to forward messages to a sink that matches a configuration based on message type, operation, agentID, and/or thingID. Routing doesn't change the message, just forward it.
The routing can dynamically change and adapt to state like time of day, load, available servers or some other relevant state.
Routing can also pass the message to multiple sinks, for example to a dispatcher and to a logging module.

## Service Modules

Service modules provide a service for use by consumers. They have their own TD describing capabilities that can be accessed using WoT messages via one of the WoT transports. Some services can have a dedicated http/rest endpoint if specified. The pipeline runtime has a built-in https server that is available to the services.

## Digital Twin Service

Status: Planned, priority high

The Digital Twin Service provides a digital replica of an IoT device. It serves its own directory with digital twin TD's. IoT device communication is intercepted and used to update the digital twin. Consumer communication is also intercepted and directed to the serve the digital twin.

### Directory Service

Status: In development, priority high

The directory service stores TD's of discovered devices and provides API's to query and update these.

WoT has specified that the TD of the directory service can be found on the discovery server ".well-known/wot" endpoint. The role of the directory service is only to supply the TD, not handle its discovery. See the discovery module for discovery support.

The directory service can be accessed through two API's. An HTTPS REST API as defined by WoT, and a Thing messaging API passed to the pipeline for read and query actions and events with update notifications.

The names of actions, events and properties of the directory service follow the [Directory Service API specification](https://w3c.github.io/wot-discovery/#exploration-directory-api).

### Logging Service

Status: planned, priority high

Logging and tracing of messages

### History Service

Status: planned, priority high

Recording of messages

## Sensor Modules

Status: planned, priority medium

The primary purpose of a sensor module is to monitor one or more sensors and send notifications when values change. Current sensor state is available as properties. Changes of key sensor values are sent as events.

Event and property subscriptions are typically not managed by the sensor module but rather by the messaging module consumers connect to.

Intended to build stand-alone IoT sensor devices or build sensor capabilities into an existing device.

## Actuator Modules

Status: planned, priority medium

The primary purpose of an actuator module is to control and monitor actuators and send notifications when the actuator state changes.

Event and property subscriptions are typically not managed by the actuator module but rather by the messaging module consumers connect to.

Intended to build stand-alone IoT actuator devices or build actuator capabilities into an existing device.

## Controller Modules

Status: planned, priority high

Controller modules interact with many sensors and actuators via a controller that operates an IoT protocol such as zwave, zigbee or other. A controller module translates between native protocol and the sensor or actuator these devices represent. Controller Modules generate a sensor or actuator module internally for each of the devices it has access to.

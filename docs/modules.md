# Pipeline Modules

This document provides an overview of available or planned HiveKit pipeline modules.

## Messaging Modules

The role of a messaging module is to convert a message between two formats. For example, a WoT WSS module can receive WoT compatible websocket messages over TLS and converts them to the module pipeline message format and returns a response in the WoT WSS format.

Various messaging protocols are supported. More are planned in the future.

### UDS IPC Messaging Client and Server Modules

This internal messaging system uses an UDS based IPC protocol to pass messages from client to server. The IPC protocol supports responses to be returned to the sender.
IPC protocols can be based on shared memory, named pipes, or unix sockets.

### WoT Websocket Messaging Client and Server Modules

WoT has recently (2025) released a Websocket standard for sending WoT messages. This can be used to exchange messages between modules, or serve a WoT compatible interface to WoT consumers.

### WoT Http Messaging Client and Server Modules

The HTTP client and server modules pass messages uni-directionally. Since HTTP doesn't support return channels, it is limited to operations that don't need a response. Event subscription for example is not supported. The http module is based on the WoT HTTP protocol. Other HTTP based protocols can easily be supported.

### WoT Sse Messaging Client and Server Modules

The HTTP/SSE client and server modules send messages using HTTP and return messages using SSE for the return channel. The http/sse module is based on the HiveOT HTTP/SSE protocol. Other HTTP/SSE based protocols can easily be supported, for example the WoT SSE protocol.

### Mqtt Messaging Client and Server Modules

The MQTT client and server modules accept bi-directional messages send using the MQTT messaging protocol. Multiple variations exist that can be supported.

### CoAP Messaging Client and Server Modules

### LoRaWan Messaging Client and Server Modules

The LoRaWan module supports bi-directional messages using the LoRa network.

## Service Modules

Service modules provide a service for use by consumers. They have their own TD describing capabilities that can be accessed using WoT messages via one of the WoT transports. Some services can have a dedicated http/rest endpoint if specified. The pipeline runtime has a built-in https server that is available to the services.

### Directory Service

The directory service stores TD's of discovered devices and provides API's to query and update these.

WoT has specified that the TD of the directory service can be found on the discovery server ".well-known/wot" endpoint. The role of the directory service is only to supply the TD, not handle its discovery. See the discovery module for discovery support.

The directory service can be accessed through two API's. An HTTPS REST API as defined by WoT, and a Thing messaging API passed to the pipeline for read and query actions and events with update notifications.

The names of actions, events and properties of the directory service follow the [Directory Service API specification](https://w3c.github.io/wot-discovery/#exploration-directory-api).

### Logging Service

Logging of messages

### History Service

Recording of messages

## Sensor Modules

The primary purpose of a sensor module is to monitor one or more sensors and send notifications when values change. It generate a TD document that describes the sensor capabilities, configuration, attributes, and events. This can come in various types.

Event and property subscriptions are typically not managed by the sensor module but rather by the messaging module consumers connect to.

### Direct Sensors

Direct sensor modules interact directly with the host system and publish TD, events, properties messages accordingly.

## Actuator Modules

The primary purpose of an actuator module is to monitor an actuator and send notifications when the value changes. It also generates a TD document that describes the actuator capabilities, configuration, attributes, events and supported actuator actions.

Similar to sensor modules an actuator module comes in the form of direct actuator and controlled actuators.

## Controller Modules

Controller modules interact with many sensors and actuators via an intermediary protocol such as zwavejs, zigbee or other. A controller module translates between native protocol via the controller device and the sensor or actuator these devices represent. Controller Modules generate a sensor or actuator module internally for each of the devices it controls.

## Routing Module

The primary purpose of a routing module is to forward messages to a sink that matches a configuration based on message type, operation, agentID, and/or thingID. Routing doesn't change the message, just forward it.
The routing can dynamically change and adapt to state like time of day, load, available servers or some other relevant state.
Routing can also pass the message to multiple sinks, for example to a dispatcher and to a logging module.

## Dispatcher Module

The primary purpose of a dispatcher module is to provide the handler for a message. The handler can also be another module. The mapping between message and handler can be statically or dynamically configured.

## Filter Module

The primary purpose of a filter module is to forward messages that match the filter parameters, based on message type, device type, operation, agentID, thingID and/or property/event/action name. It can for example be used to determine what messages are passed to a logger.

## Authorization Module

The primary purpose of an authorization modules is to authorize a message to be forwarded, based on the authenticated sender, operation, agentID, thingID and/or property or action name.

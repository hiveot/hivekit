# HiveKit - HiveOT Development Kit

HiveKit provides modules for building lightweight IoT applications for integration with the Web of Things.

HiveKit is not an application but intended to offer the building blocks to easily construct IoT applications.

The concept is that an application is build by combining modules that each provides a needed capability. Interactive modules define their capabilities using a W3C Thing Description (TD) document. Modules are linked in a chain. Each module handles request messages directed at their thingID. Modules emit notifications for events and property updates.

The standard module has a simple interface: A handler for request messages with a replyTo callback, and a handler for notification messages. Modules are linked by setting a request sink to the next module in the chain. Similarly a notification sink is set to the upstream module.

[![module](docs/hivekit-module.png)](#hivekit-modules)

## Project Status

HiveKit is in alpha development (June 2026).

Most modules are implemented in golang. Javascript and Python integration is planned. Using transport modules it is easy to link Javascript, Python and golang modules with minimal overhead.
Modules with a checkmark are functional but breaking changes can still be expected for those marked as alpha or beta.

Core Service modules:

| status | module      | description                        | stage |
| :----: | ----------- | ---------------------------------- | ----- |
|   ✔️    | agent       | Producer of IoT data               | alpha |
|   ✔️    | authn       | Client authentication              | alpha |
|   ✔️    | authz       | Role based authorization           | alpha |
|   ✔️    | bucketstore | Key-value data storage             | alpha |
|   ✔️    | certs       | Certificate management             | alpha |
|   ✔️    | consumer    | Consumer of IoT data               | alpha |
|   ✔️    | digitwin    | Digital twin                       | alpha |
|   ✔️    | directory   | Thing directory                    | alpha |
|   ✔️    | factory     | Module factory                     | alpha |
|   ✔️    | history     | Message history recorder           | alpha |
|   ✔️    | logging     | Basic messaging logging            | alpha |
|   ✔️    | reconnect   | Restore dropped client connections | alpha |
|   ✔️    | router      | Message routing to remote devices  | alpha |
|   ✔️    | vcache      | Value cache                        | alpha |
|   ⬛    | jsscript    | Javascript based automation        | todo  |
|   ⬛    | rules       | Rule based automation              | todo  |

[Transport modules](docs/transport.md):

Transport modules come with a server and a client module.

| status | module                  | description                           | stage |
| :----: | ----------------------- | ------------------------------------- | ----- |
|   ✔️    | transport/discovery     | WoT mDNS device discovery             | alpha |
|   ✔️    | transport/grpc          | HiveOT gRPC fast message streaming    | alpha |
|   ✔️    | transport/httpbasic     | WoT HTTP basic messaging protocol     | alpha |
|   ✔️    | transport/httptransport | HTTP server for sub-protocols         | alpha |
|   ✔️    | transport/ssesc         | HiveOT HTTP/SSE-SC messaging protocol | alpha |
|   ✔️    | transport/wss           | WoT Websocket messaging protocol      | alpha |
|   ⬛    | transport/mqtt          | WoT MQTT messaging protocol           | n/a   |

Integration Binding Modules:

| status | module   | description                     | stage |
| :----: | -------- | ------------------------------- | ----- |
|   ⬛    | ipnet    | IP Network monitor              | todo  |
|   ⬛    | isy99x   | ISY 99 gateway binding          | todo  |
|   ⬛    | owserver | 1-wire owserver gateway binding | todo  |
|   ⬛    | zwavejs  | ZWave binding using zwave-js    | todo  |
|   ⬛    | weather  | Weather service bindings        | todo  |
|   ⬛    | lorawan  | LoRaWan gateway binding         | todo  |
|   ⬛    | canbus   | Canbus gateway binding          | todo  |
|   ⬛    | ...      | and many more...                | todo  |

## HiveKit Modules

HiveKit modules are building blocks for building devices and applications. Modules follow the separation of concerns paradigm where each module is performs a single task. Applications are build by combining modules. 

Individual modules are also Things and identified by their instance thing-ID. Where applicable, their capabilities can be described by a WoT TD (Thing Description) document that describes its properties, events and actions. Interaction takes place by creating a RequestMessage with an operation and the module ThingID and sending it to the module.

A [HiveKit module](hivekit-module.png) MUST implement the IHiveModule interface. This interface governs the interaction with the module and enables the ability to add their functionality to a chain of modules.

The IHiveModule interface describes how to link a module to the next module in the chain. The link consists of a request handler to pass request messages down the chain and respond with a response message, and a notification handler to pass notification messages up the chain. A 'HiveModuleBase' helper is available that implements this interface and supports linking of modules. HiveModuleBase is used by most HiveKit modules.

HiveKit modules interact using _RRN_ Request-Response and publish-subscribe Notification messages. HiveKit combines the strengths of these two messaging patterns into a simple and easy to use messaging framework for connecting modules. RRN messages define an envelope that describes a WoT operation, the Thing to address, the name of the message, and its payload, as described in the [W3C WoT Thing Description](https://www.w3.org/TR/wot-thing-description11/).

### Module API

All modules support the HiveKit module API defined as IHiveModule. This API defines how to handle requests and responses.


```go
// The golang HiveOT module interface. The JS and Python implementation will offer something similar.
type IHiveModule interface {

	// GetThingID returns the module's instance ID.
	GetThingID() string

	// Handle the notification received from a producer.
	// The default behavior is to forward it upstream to the handler set with SetNotificationSink.
	HandleNotification(notif *msg.NotificationMessage)

	// HandleRequest processes or forwards a request downstream.
	HandleRequest(request *RequestMessage, replyTo(resp *ResponseMessage)) error

	// Set the handler of notifications emitted by this module
	SetNotificationSink(consumer IHiveModule)

	// SetRequestSink sets the handler of requests emitted by this module.
	SetRequestSink(sink IHiveModule)

	// Start readies the module for use
	Start() error
	Stop()
}
```

### Module Types

There are two fundamental types of modules, producers and consumers of information. Producers handle requests and publish information while consumers publish requests and receive notifications. An IoT device is typically a producer while an end-user interacts using a consumer module. In between a producer and consumer there can be many other modules at work that act as a producer, consumer or both.


The following module categories can be distinguished:

1. Service modules are producers that offer a service, such as authentication, logging and routing. Service modules can be configured through properties and queried using actions.

The 'Agent' module implementation helps writing producers. It provides methods for publishing notifications, tracking state and handle requests to read properties.

2. Middleware modules are a class of modules whose purpose is to analyze, filter and route messages. For example, logging, authorizing, routing are middleware tasks. These modules act as producers for consumers and consumers for producers.

3. Transport modules role is to link modules over the network. They come in two flavors, a transport client and a transport server module. The client module sends requests to the server and the server module sends requests and notifications to the client. Client-Server module pairs are available for multiple protocols such as http-basic, websockets, gRPC and others. Server modules track event subscriptions and subscriptions to observe properties made via the client.

4. Consumer modules collect information from producers. Consumers publish requests for information and receive responses and notifications. Services that aggregate, transform or enrich information are consumers of that information. A user interface for example is a consumer that presents information. 
   
The 'Consumer' module implementation helps writing consumers by providing methods for publishing requests and subscribing to event and property notifications.

## Linking Modules

A core capability of modules is the ability to chain them together. Chains offer application level functionality. A chain can operate on a single computer system or include modules across multiple computer systems linked by transport modules. This allows for creating a powerful distributed IoT solution with small lightweight modules that require few resources and are simple to maintain.

Creating a module chain can be done manually by programatically linking modules, or dynamically by providing a recipe to the factory service. 

### Module Factory

Modules in HiveKit are not applications themselves but intended to construct an application. The [factory module](go/modules/factory/README.md) facilitates building applications by chaining modules defined in a recipe. This chaining aggregates functionality provided by each module. 

Application specific logic can easily be incorporated using the hooks provided by the agent module, or by providing application logic as a module itself and adding this module to the recipe.

![module](docs/module-chain.png)


### Adding Modules

One of the goals of HiveKit is to make it easy to add compatible modules.

To develop a module implement its IHiveModule interface. The provided ModuleBase implements the little boilerplate that is needed. The HandleRequest method is the most important method to implement. Exposing a TM is recommended for IoT devices.

To use a module connect it as the sink of the previous module in the chain. In case of an IoT device the previous module can be one of the messaging server modules. The server passes requests to the HandleRequest method which the module must implement, and responses are returned to the sender. Notifications emitted by the module are passed to the registered notification handler which is the server module.


## About HiveOT

Security is big concern with today's IoT devices. The Internet of Things contains billions of devices that when not properly secured can be hacked too easily. Unfortunately the reality is that the security of many of these devices leaves a lot to be desired. Many devices are vulnerable to attacks and are never upgraded with security patches. This problem is only going to get worse as more IoT devices are coming to market. A botnet of a billion IoT devices can bring parts of the Internet to its knees and cripple essential services. The cost to businesses and consumers reaches hundreds of millions of dollars yearly.

Exposing IoT devices to the internet for direct use by consumers is therefore simply a very very bad idea from a security point of view, and does not meet the needs of todays reality. And yet, for some reason every year more and more IoT devices hit the market that run their own server and are exposed to the internet.

While HiveKit lets you build individual IoT devices that run their own server (please don't), it should be clear by now that this is, well ..., a very very bad idea.

HiveOT aims to aid in improving security of the IoT ecosystem by:

1. Not run a server on IoT devices. Instead IoT devices connect to a secured gateway or hub. These devices have the RC (reverse connection) capability which is readily supported by all HiveKit transport modules. Just swap a server module for its client counterpart.
2. Offer an easy way to build a gateway or hub that supports RC capable devices. This is equivalent to building a server that forwards request to connected clients using the router module.
3. Support an easy way to expand the application functionality with custom modules without having to be a security expert.
4. Support the W3C WoT standard for interacting with IoT devices including authentication, authorization, directory, history and other capabilities.
5. Define a development commitment (see below) when using HiveOT software.

HiveOT is based on the [W3C WoT TD 1.1 specification](https://www.w3.org/TR/wot-thing-description11/) for interaction between IoT devices and consumers. It aims to be compatible with this standard.

Integration with 3rd party IoT protocols is supported through the use of protocol binding modules. These modules translate between the 3rd party IoT protocols and RRN (request/response/notification) messages. The RRN messages can be linked to a WoT protocol for interaction with WoT compatible clients using properties, events and actions.

## Developer Commitment

This project is aimed at software developers for building secure IoT solutions. When adopting HiveKit, developers agree to:

1. Support the security mandate that individual IoT devices should remain isolated from the internet. See above for the motivation and rational of this critical aspect.
2. Support the use of RC (reverse connection) enabled devices that connect to a secured gateway or hub. When possible, promote this approach with the WoT working group.
3. Agree to regularly provide security fixes with firmware updates if needed.

This probably needs a modified MIT license but that is beyond the scope of this project.

## Getting Started

### Build

This project uses golang 1.25 or newer.

To debug with vscode delve must be installed. To get the latest (on linux):

> go install github.com/go-delve/delve/cmd/dlv
> export $PATH=$PATH:~/go/bin
> go mod tidy

### Use

The easiest way to get started is to use the factory module with one of the example recipes. There are recipes for constructing stand-alone IoT devices, a WoT compatible gateway, a digital twin hub, and client applications. [see factory for details](go/modules/factory/README.md)

... this section is under development...

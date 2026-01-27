# HiveOT Development Kit

HiveKit provide building blocks for constructing IoT applications.

[Overview](hivekit.png)

## Status

HiveKit is in early development (Jan 2026).

It is reworked from a library into a collection of reusable modules for easy construction of IoT applications using module pipelines. An outline of this concept is described below.

HiveKit is not an application but intended to offer the building blocks to easily construct IoT applications. It is used in the [HiveFlow](https://github.com/hiveot/hiveflow) and [HiveHub](https://github.com/hiveot/hub) applications. (currently being reworked)

## HiveKit Modules

HiveKit building blocks are called 'modules'.

Anything that implements the IHiveModule interface is usable as a HiveKit module. The purpose of the module interface is to standardize the interaction with the module through Request-Response and publish-subscribe Notification messages (RRN). HiveKit combines the strengths of these two messaging patterns into a simple and easy to use messaging framework for module interaction.

RRN messages define an envelope that describes an operation, Thing, and name of the message along with its payload, following the WoT standard.

All modules are identified by their unique module instance ID. Messages can be targeted to a specific module by using the module ID as the thingID. A module can be published as a WoT Thing that can be configured and controlled using messages. To this end it the module API contains a method to export its Thing TM that describes the module properties, events and actions.

Examples of modules:

- directory, history, authentication services.
- a message processor such as a router, filter, logger, etc
- transport protocol bindings such as WoT HTTP, WSS, CoAP, MQTT
- an IoT adapter such as ZWave, Insteon, CoAP, Zigbee, Shelley and so on.
- a consumer

While HiveKit comes with a set of ready to use modules, 3rd party modules can be incorporated easily as part of a recipe.

Last but not least, modules can be distributes across multiple systems to act as parts of a whole solution.

## Types of Modules

There are two fundamental types of modules, producers and consumers of information. Modules can act as both types. An IoT device is typically a producer while an end-user interacts using a consumer module. In between a producer and consumer there can be many other modules at work that act as a producer, consumer or both.

Producer modules generate IoT information, typically obtained from sensors, actuators or other means. Producers handle requests for information and requests for actions, respond with the results, and publish notifications of events and updates. Services that publish aggregated, transformed, or enriched information are also producers.

Consumer modules collect information from producers. Consumers publish requests for information and receive responses and notifications. Services that aggregate, transform or enrich information are consumers of that information. A user interface for example is a consumer that presents information.

Services are often both a consumer and producer of information. A history service consumes information to store it. It is also a producer to allow retrieval of the stored information. A rules based automation service is a consumer of information that is used in the rules, and a producer of information when rules are triggered. Internally, services should be split into consumer and producer parts, maintaining a separation of concerns.

Transport modules are a class of modules whose purpose is to transport a request from consumer to producer, return the response, and transport notifications from producers to subscribed consumers without modifying the information. Most transport module offer two submodules, a client and a server. The distinction between a client and server module is purely in who initiates a connection and who serves the connection. Producers and consumers can exist on either side of a connection. This implies that a both sides of transport connection can send and receive requests, responses and notifications. Transport modules therefore act as producers for consumers, and consumers for producers.

Middleware modules are a class of modules whose purpose is to analyze, filter and route messages. For example, logging, authorizing, routing are middleware tasks. These modules act as producers for consumers and consumers for producers.

## Information flows

The information flows between producers and consumers are the request flow, response flow and notification flow:

- The request flow consists of request messages flowing from consumers to producers.
- The response flow consists of response messages flowing from producers to consumers.
- The notification flow consists of notification messages flowing from producers to consumers.

This is represented in the module API that must accomodate these flows between modules.

When a consumer publishes a request, the request has to be accepted by a module acting as a producer. Thus a consumer always links to a producer and vice-versa.

Transport and middleware modules thus link with consumers acting as a producer and link with producers acting as a consumer.

### Producer

To act as a producer, the module must implement a HandleRequest call. When receiving a request it must process it and send a response to the provided callback. The response can be send asynchronously at any time, even before HandleRequest returns. If the request cannot be processed it must be forwarded to the registered sink. If it cannot be forwarded an error must be returned indicating that the request cannot be fulfilled.

Producers publish notifications. There are two criteria for receiving these notifications, a subscription and a callback handler. HiveOT puts the subscription handling at the inter-process transport boundary (eg, protocol binding), therefore transport modules handle subscription request messages. This aligns with WoT that defines subscription requests as part of the protocol binding, and with pub/sub protocols such as MQTT where subscription is handled by the protocol server. Producers themselves therefore do not need to implement subscription handling.

Middleware modules act as a producer that simply forward requests until a producer is received that can process the request.

### Consumer

When acting as a consumer, a module publishes a request to a producer. It therefore must link to a producer so it can invoke its HandleRequest. The API for this is: SetSink(producer).

Consumers subscribe to notifications by publishing a Subscribe request message. This is passed down the module chain until a transport module is received that registers the subscription. If no module in the chain handles subscriptions then this fails with an error.

Consumers can receiving notifications from producers that reside in-process by registering a notification callback handler with the producer using its SetNotificationHandler. Note that a module API only accepts a single notification callback handler. This callback receives all notifications. It is up to the consumer to filter the desired notification.

Since consumers can act as a producer as part of a chain (most do) a received notification can be passed up the pipeline by using the callback registered by the previous consumer in the chain. Unless the chain is long this should be fairly efficient. If the chain is long with many in-process modules then a helper module can be used that acts as a router for notifications. This notification router is a potential future module that will be added when the need arises.

As consumers can must also receive asynchronous notifications. This is accomplished by invoking a Subscribe request on the sink, providing a callback with the notification messages.

## Anatomy Of A Module

A module can be broken down into separate responsibilities, each can be implemented with little code and effort. A simple base class implements the boilerplate for the bulk of this. The core business logic and conversion between native API and RRN messages must be coded however.

1. The module factory provides a standard way to create and use modules through provided configuration. It provides a factory function to create module instances from configuration and optionally serves the TM and possibly other formats such as REST and protobuf definitions.

2. The module core implements the module's business logic and can be used as-is without HiveOT specific dependencies. The core must implement the native api as defined in the module api section.

3a. The messaging request server receives requests and converts them to a native API call for the core. Responses are encoded into a standard RRN response message envelope and passed to the provided callback.

When requests messages are addressed to a Thing not managed by the module then the request is simply forwarded to next producer in the chain, set with 'SetSink'.

3b. Optionally additional messaging handlers, such as http server handlers, can are included if the module supports other protocols such as HTTP/REST.

4. The subscription server tracks subscription requests and invokes the callback when a matching notification is received from the core.

Similar to the request handler, when a subscription request is addressed to a Thing not managed by the module, the request is simply forwarded to the next producer in the chain.

5. A messaging client translate from a native API to RRN messages. The client is typically paired with a transport client as its sink for delivery of request to the remote module.

5b. Optionally additional messaging clients can exists to facilitate interaction using other protocols such has REST based requests.

6. Last but not least the module tests contains test methods to verify the correct behavior of the module including its handlers and clients.

## Linking Modules Into A Pipeline

A core capability of modules is the ability to chain them together to form a pipeline. Pipelines offer application level functionality. A pipeline can operate on a single computer system or include modules across multiple computer systems. This allows for creating a powerful distribute IoT solution with small lightweight modules that require few resources and are simple to maintain.

A pipeline is described by a recipe. The recipe defines which modules are used and how they are linked using their RRN message interface. It is intended to simplify building and testing WoT compatible IoT devices and to provide reusable capabilities for processsing IoT device information.

Creating pipelines can be done manually by programatically linking modules, or dynamically by providing a recipe to the pipeline service. A dynamic pipeline can be reconfigured in a live system to adapt to changing needs of the environment based on the circomstances.

When two modules are linked, the first module is called the consuming module and the module it attaches to is called the producer module. The consuming module sends requests, receives responses and subscribes to notifications. The producer module figures out how to resolve the request it received and returns the response and notifications.

In the linking process, the consuming module simply registers the producer module it is linked to as its sink.

Modules can be linked in several ways:

1. At application compile time the application creates the module instances and sets the producers as sinks to the consumers in the desired order, creating a pipeline. In golang this creates a single binary that can run as a stand-alone application. This is made available in phase 1.
2. Using the pipeline service runtime to link modules based on the provided yaml recipe. The pipeline recipe can be changed without the need to recompile. This is limited to the use of modules available in the pipeline runtime. This is phase 2.
3. By compiling a pipeline using the provided yaml recipe using the hiveflow CLI. In golang this creates a single binary that has its recipe and required modules embedded. This effectively generates an application at runtime using the given recipe. This is made available in phase 3.

Using messaging modules the pipeline can be linked to other modules running elsewhere making a distributed application.

Testing is facilitated by providing a testing runtime with tools to generate devices, consumers and messages. Modules can be tested in isolation or as part of a pipeline.

### HiveOT Module API

All (sub)modules support the HiveOT module API defined as IHiveModule. This API defines how to handle requests, responses and subscribe to notifications. Depending on the programming language static or dynamic linking can be used to implement the callback hooks.

```go
// The golang HiveOT module interface. The JS and Python implementation will offer something similar.
type IHiveModule interface {
  	// GetModuleID returns module's ID.
	GetModuleID() string

	// GetTM returns the module's [W3C WoT Thing Model](https://www.w3.org/TR/wot-thing-description11/#thing-model)
	GetTM() string

	// HandleRequest [producer] processes or forwards a request downstream.
	HandleRequest(request *RequestMessage, replyTo(resp *ResponseMessage) error

	// Set the handler of notifications produced (or forwarded) by this module
	SetNotificationHandler(consumer msg.NotificationHandler)

	// SetSink set the given module as the producer for requests and notifications.
	SetSink(producer IHiveModule)

	// Start readies the module for use using the given yaml configuration.
	Start(yamlConfig string) error
	Stop()
}
```

## Adding Modules

One of the goals of HiveKit is to make it easy to add compatible modules.

To develop a module implement its IHiveModule interface. The provided ModuleBase implements the little boilerplate that is needed. The HandleRequest method is the most important method to implement. Exposing a TM is recommended for IoT devices.

To use a module connect it as the sink of the previous module in the chain. In case of an IoT device the previous module can be one of the messaging server modules. The server passes requests to the HandleRequest method which the module must implement, and responses are returned to the sender. Notifications emitted by the module are passed to the registered notification handler which is the server module.
